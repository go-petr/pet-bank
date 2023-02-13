package middleware

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-petr/pet-bank/pkg/configpkg"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

type RequestIDKey struct{}

func GetLogger(config configpkg.Config) zerolog.Logger {

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	var (
		output   io.Writer = os.Stderr
		logLevel           = zerolog.InfoLevel // default to INFO
	)

	log := zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Logger()

	if config.Environement == "development" {
		log = log.
			Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).
			Level(zerolog.TraceLevel).
			With().
			Caller().
			Logger()
	}

	return log
}

func requestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(RequestIDKey{}).(string)
	if !ok {
		return "-"
	}
	return requestID
}

// RequestLogger logs a gin HTTP request in JSON format.
func RequestLogger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		start := time.Now()

		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
			c.Request.Header.Set("X-Request-ID", requestID)
			c.Writer.Header().Set("X-Request-ID", requestID)
		}

		logger = logger.With().Str("request_id", requestID).Logger()

		c.Request = c.Request.WithContext(logger.WithContext(c.Request.Context()))

		// Process request
		c.Next()

		defer func() {

			if panicVal := recover(); panicVal != nil {
				logger.Error().Msgf("panic message: %v", panicVal)
				c.Writer.WriteHeader(http.StatusInternalServerError)
			}

			// Fill the params
			param := gin.LogFormatterParams{}
			param.TimeStamp = time.Now() // Stop timer
			param.Latency = param.TimeStamp.Sub(start)
			param.ClientIP = c.ClientIP()
			param.Method = c.Request.Method
			param.StatusCode = c.Writer.Status()
			param.ErrorMessage = c.Errors.ByType(gin.ErrorTypePrivate).String()
			param.Path = c.Request.URL.Path

			var logEvent *zerolog.Event
			if c.Writer.Status() >= 500 {
				logEvent = logger.Error()
			} else {
				logEvent = logger.Info()
			}

			logEvent.
				Str("client_id", param.ClientIP).
				Str("method", param.Method).
				Int("status_code", param.StatusCode).
				Str("path", param.Path).
				Str("latency", param.Latency.String()).
				Msg(param.ErrorMessage)
		}()
	}
}

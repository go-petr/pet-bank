// Package web defines common components for a web application.
package web

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// Error wraps a given err into json frinedly struct.
func Error(err error) Response {
	return Response{Error: err.Error()}
}

// Response holds the common response type for all APIs.
type Response struct {
	AccessToken           string    `json:"access_token,omitempty"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at,omitempty"`
	RefreshToken          string    `json:"refresh_token,omitempty"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at,omitempty"`
	Data                  any       `json:"data,omitempty"`
	Error                 string    `json:"error,omitempty"`
}

// GetErrorMsg parses error message from request validator.
func GetErrorMsg(ve validator.ValidationErrors) string {

	field := ve[0]
	errMsg := field.Field()

	switch field.Tag() {
	case "required":
		errMsg += " field is required"
	case "lte":
		errMsg += " must be less than " + field.Param()
	case "gte":
		errMsg += " must be greater than " + field.Param()
	case "alphanum":
		errMsg += " accepts only alphanumeric characters"
	case "min":
		errMsg += " must be at least " + field.Param() + " characters long"
	case "max":
		errMsg += " must be less than " + field.Param()
	case "email":
		errMsg += " must contain a valid email"
	case "currency":
		errMsg += " is not supported"
	default:
		errMsg += " unknown error"
	}

	return errMsg
}

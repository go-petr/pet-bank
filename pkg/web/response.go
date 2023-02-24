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
func GetErrorMsg(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return " field is required"
	case "lte":
		return " must be less than " + fe.Param()
	case "gte":
		return " must be greater than " + fe.Param()
	case "alphanum":
		return " accepts only alphanumeric characters"
	case "min":
		return " must be at least " + fe.Param() + " characters long"
	case "email":
		return " must contain a valid email"
	case "currency":
		return " is not supported"
	}

	return "Unknown error"
}

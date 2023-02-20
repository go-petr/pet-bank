// Package web defines common components for a web application.
package web

// JSONError provides type for explicit json encoded error response.
type JSONError struct {
	Error string `json:"error"`
}

// Error wraps a given err into json frinedly struct.
func Error(err error) JSONError {
	return JSONError{Error: err.Error()}
}

// Response holds the common response type for all APIs.
type Response struct {
	AccessToken           string    `json:"access_token,omitempty"`
	AccessTokenExpiresAt  string    `json:"access_token_expires_at,omitempty"`
	RefreshToken          string    `json:"refresh_token,omitempty"`
	RefreshTokenExpiresAt string    `json:"refresh_token_expires_at,omitempty"`
	Data                  any       `json:"data,omitempty"`
	Error                 JSONError `json:"error,omitempty"`
}

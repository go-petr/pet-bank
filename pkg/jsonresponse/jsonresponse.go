// Package jsonresponse enables consistent responses across all handlers.
package jsonresponse

// JSONError provides type for explicit json encoded error response.
type JSONError struct {
	Error string `json:"error"`
}

// Error wraps a given err into json frinedly struct.
func Error(err error) JSONError {
	return JSONError{Error: err.Error()}
}

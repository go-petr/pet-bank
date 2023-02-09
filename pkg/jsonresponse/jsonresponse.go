// Package jsonresponse enables consistent responses across all handlers.
package jsonresponse

// jsonError provides type for explicit json encoded error response.
type jsonError struct {
	Error string `json:"error"`
}

// Error wraps a given err into json frinedly struct.
func Error(err error) jsonError {
	return jsonError{Error: err.Error()}
}

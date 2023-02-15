// Package errorspkg provides common app errors.
package errorspkg

import "errors"

// ErrInternal indicates internal server error.
var ErrInternal = errors.New("internal")

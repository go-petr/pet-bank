// Package tokenpkg implements common token makers.
package tokenpkg

import "time"

// Maker is an interface for managing tokens
//
//go:generate mockgen -source maker.go -destination maker_mock.go -package tokenpkg
type Maker interface {
	// CreateToken creates a new token for a specific username and duration
	CreateToken(username string, duration time.Duration) (string, *Payload, error)

	// VerifyToken checks if the token is valid or not
	VerifyToken(token string) (*Payload, error)
}

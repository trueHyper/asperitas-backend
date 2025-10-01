package session

import "time"

type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Repository interface {
	Create(userID, sessionID string) (string, error)
	IsValid(userID string) (bool, error)
	Invalidate(userID string) error
}

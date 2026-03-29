package models

import "time"

// CreateUserParams is a persistence boundary shape for inserting users.
type CreateUserParams struct {
	ID        string
	Username  string
	PINHash   string
	CreatedAt time.Time
}

// CreateSessionParams is a persistence boundary shape for inserting sessions.
type CreateSessionParams struct {
	Token     string
	UserID    string
	CreatedAt time.Time
}

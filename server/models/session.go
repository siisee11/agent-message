package models

import "time"

// Session represents an opaque bearer token owned by a user.
type Session struct {
	Token     string    `json:"token" db:"token"`
	UserID    string    `json:"user_id" db:"user_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

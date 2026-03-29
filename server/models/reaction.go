package models

import "time"

// Reaction is an emoji response to a message by a user.
type Reaction struct {
	ID        string    `json:"id" db:"id"`
	MessageID string    `json:"message_id" db:"message_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Emoji     string    `json:"emoji" db:"emoji"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

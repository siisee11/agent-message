package models

import "time"

// Conversation is a direct-message pairing between two users.
type Conversation struct {
	ID           string    `json:"id" db:"id"`
	ParticipantA string    `json:"participant_a" db:"participant_a"`
	ParticipantB string    `json:"participant_b" db:"participant_b"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

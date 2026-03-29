package models

import "time"

// User is the persisted account record.
type User struct {
	ID        string    `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	PINHash   string    `json:"-" db:"pin_hash"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// UserProfile is the safe user projection for API responses.
type UserProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func (u User) Profile() UserProfile {
	return UserProfile{
		ID:        u.ID,
		Username:  u.Username,
		CreatedAt: u.CreatedAt,
	}
}

package models

import (
	"strings"
	"time"
)

// User is the persisted account record.
type User struct {
	ID           string    `json:"id" db:"id"`
	AccountID    string    `json:"account_id" db:"account_id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// UserProfile is the safe user projection for API responses.
type UserProfile struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func (u User) EffectiveUsername() string {
	if username := strings.TrimSpace(u.Username); username != "" {
		return username
	}
	return strings.TrimSpace(u.AccountID)
}

func (p *UserProfile) ApplyUsernameFallback() {
	if p == nil {
		return
	}
	if strings.TrimSpace(p.Username) != "" {
		return
	}
	p.Username = strings.TrimSpace(p.AccountID)
}

func (u User) Profile() UserProfile {
	return UserProfile{
		ID:        u.ID,
		AccountID: u.AccountID,
		Username:  u.EffectiveUsername(),
		CreatedAt: u.CreatedAt,
	}
}

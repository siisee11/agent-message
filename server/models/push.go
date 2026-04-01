package models

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrPushSubscriptionEndpointRequired = errors.New("endpoint is required")
	ErrPushSubscriptionP256DHRequired   = errors.New("p256dh key is required")
	ErrPushSubscriptionAuthRequired     = errors.New("auth key is required")
)

type PushSubscription struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Endpoint  string    `json:"endpoint" db:"endpoint"`
	P256DH    string    `json:"p256dh" db:"p256dh"`
	Auth      string    `json:"auth" db:"auth"`
	UserAgent string    `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type WebPushKeys struct {
	P256DH string `json:"p256dh"`
	Auth   string `json:"auth"`
}

type UpsertPushSubscriptionRequest struct {
	Endpoint string      `json:"endpoint"`
	Keys     WebPushKeys `json:"keys"`
}

func (r UpsertPushSubscriptionRequest) Validate() error {
	if strings.TrimSpace(r.Endpoint) == "" {
		return ErrPushSubscriptionEndpointRequired
	}
	if strings.TrimSpace(r.Keys.P256DH) == "" {
		return ErrPushSubscriptionP256DHRequired
	}
	if strings.TrimSpace(r.Keys.Auth) == "" {
		return ErrPushSubscriptionAuthRequired
	}
	return nil
}

type DeletePushSubscriptionRequest struct {
	Endpoint string `json:"endpoint"`
}

func (r DeletePushSubscriptionRequest) Validate() error {
	if strings.TrimSpace(r.Endpoint) == "" {
		return ErrPushSubscriptionEndpointRequired
	}
	return nil
}

type PushConfigResponse struct {
	Enabled        bool   `json:"enabled"`
	VAPIDPublicKey string `json:"vapid_public_key,omitempty"`
}

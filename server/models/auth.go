package models

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrUsernameRequired = errors.New("username is required")
	ErrPINInvalid       = errors.New("pin must be 4-6 digits")
	ErrTokenRequired    = errors.New("token is required")
)

var pinPattern = regexp.MustCompile(`^\d{4,6}$`)

// RegisterRequest is the JSON body for POST /api/auth/register.
type RegisterRequest struct {
	Username string `json:"username"`
	PIN      string `json:"pin"`
}

func (r RegisterRequest) Validate() error {
	return validateCredentials(r.Username, r.PIN)
}

// LoginRequest is the JSON body for POST /api/auth/login.
type LoginRequest struct {
	Username string `json:"username"`
	PIN      string `json:"pin"`
}

func (r LoginRequest) Validate() error {
	return validateCredentials(r.Username, r.PIN)
}

// AuthResponse is returned by successful register/login endpoints.
type AuthResponse struct {
	Token string      `json:"token"`
	User  UserProfile `json:"user"`
}

func (r AuthResponse) Validate() error {
	if strings.TrimSpace(r.Token) == "" {
		return ErrTokenRequired
	}
	return nil
}

func validateCredentials(username, pin string) error {
	if strings.TrimSpace(username) == "" {
		return ErrUsernameRequired
	}
	if !pinPattern.MatchString(pin) {
		return ErrPINInvalid
	}
	return nil
}

package models

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrUsernameRequired = errors.New("username is required")
	ErrUsernameInvalid  = errors.New("username may contain only letters, numbers, dot, underscore, and hyphen")
	ErrUsernameLength   = errors.New("username must be 3-32 characters")
	ErrPINInvalid       = errors.New("pin must be 4-6 digits")
	ErrTokenRequired    = errors.New("token is required")
)

const (
	UsernameMinLength = 3
	UsernameMaxLength = 32
)

var (
	pinPattern      = regexp.MustCompile(`^\d{4,6}$`)
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

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
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if !pinPattern.MatchString(pin) {
		return ErrPINInvalid
	}
	return nil
}

// ValidateUsername ensures username inputs conform to API constraints.
func ValidateUsername(username string) error {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return ErrUsernameRequired
	}
	if trimmed != username {
		return ErrUsernameInvalid
	}
	if len(trimmed) < UsernameMinLength || len(trimmed) > UsernameMaxLength {
		return ErrUsernameLength
	}
	if !usernamePattern.MatchString(trimmed) {
		return ErrUsernameInvalid
	}
	return nil
}

// ValidateUsernameQuery allows prefix search terms for username lookup endpoints.
func ValidateUsernameQuery(query string) error {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return ErrUsernameRequired
	}
	if len(trimmed) > UsernameMaxLength {
		return ErrUsernameLength
	}
	if !usernamePattern.MatchString(trimmed) {
		return ErrUsernameInvalid
	}
	return nil
}

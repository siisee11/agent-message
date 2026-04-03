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
	ErrPasswordRequired = errors.New("password is required")
	ErrPasswordLength   = errors.New("password must be 4-72 characters")
	ErrTokenRequired    = errors.New("token is required")
)

const (
	UsernameMinLength = 3
	UsernameMaxLength = 32
	PasswordMinLength = 4
	PasswordMaxLength = 72
)

var (
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

// RegisterRequest is the JSON body for POST /api/auth/register.
type RegisterRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	LegacyPIN string `json:"pin,omitempty"`
}

func (r RegisterRequest) Validate() error {
	return validateCredentials(r.Username, firstNonEmpty(r.Password, r.LegacyPIN))
}

// LoginRequest is the JSON body for POST /api/auth/login.
type LoginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	LegacyPIN string `json:"pin,omitempty"`
}

func (r LoginRequest) Validate() error {
	return validateCredentials(r.Username, firstNonEmpty(r.Password, r.LegacyPIN))
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

func validateCredentials(username, password string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return ErrPasswordRequired
	}
	if len(password) < PasswordMinLength || len(password) > PasswordMaxLength {
		return ErrPasswordLength
	}
	return nil
}

func firstNonEmpty(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
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

package models

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrAccountIDRequired = errors.New("account_id is required")
	ErrAccountIDInvalid  = errors.New("account_id may contain only letters, numbers, dot, underscore, and hyphen")
	ErrAccountIDLength   = errors.New("account_id must be 3-32 characters")
	ErrUsernameRequired  = errors.New("username is required")
	ErrUsernameInvalid   = errors.New("username may contain only letters, numbers, dot, underscore, and hyphen")
	ErrUsernameLength    = errors.New("username must be 3-32 characters")
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
	AccountID       string `json:"account_id,omitempty"`
	Password        string `json:"password"`
	LegacyPIN       string `json:"pin,omitempty"`
	LegacyAccountID string `json:"username,omitempty"`
}

func (r RegisterRequest) Validate() error {
	accountID := r.AccountIDValue()
	if err := ValidateAccountID(accountID); err != nil {
		return err
	}
	return validatePassword(firstNonEmpty(r.Password, r.LegacyPIN))
}

// LoginRequest is the JSON body for POST /api/auth/login.
type LoginRequest struct {
	AccountID       string `json:"account_id,omitempty"`
	Password        string `json:"password"`
	LegacyPIN       string `json:"pin,omitempty"`
	LegacyAccountID string `json:"username,omitempty"`
}

func (r LoginRequest) Validate() error {
	if err := ValidateAccountID(r.AccountIDValue()); err != nil {
		return err
	}
	return validatePassword(firstNonEmpty(r.Password, r.LegacyPIN))
}

type UpdateUsernameRequest struct {
	Username string `json:"username"`
}

func (r UpdateUsernameRequest) Validate() error {
	trimmed := strings.TrimSpace(r.Username)
	if trimmed == "" {
		return nil
	}
	return ValidateUsername(trimmed)
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

func validatePassword(password string) error {
	if strings.TrimSpace(password) == "" {
		return ErrPasswordRequired
	}
	if len(password) < PasswordMinLength || len(password) > PasswordMaxLength {
		return ErrPasswordLength
	}
	return nil
}

func (r RegisterRequest) AccountIDValue() string {
	return firstNonEmpty(r.AccountID, r.LegacyAccountID)
}

func (r RegisterRequest) UsernameValue() string {
	return strings.TrimSpace(r.AccountIDValue())
}

func (r LoginRequest) AccountIDValue() string {
	return firstNonEmpty(r.AccountID, r.LegacyAccountID)
}

func firstNonEmpty(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func validateIdentifier(value string, required, invalid, length error) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return required
	}
	if trimmed != value {
		return invalid
	}
	if len(trimmed) < UsernameMinLength || len(trimmed) > UsernameMaxLength {
		return length
	}
	if !usernamePattern.MatchString(trimmed) {
		return invalid
	}
	return nil
}

func ValidateAccountID(accountID string) error {
	return validateIdentifier(accountID, ErrAccountIDRequired, ErrAccountIDInvalid, ErrAccountIDLength)
}

// ValidateUsername ensures username inputs conform to API constraints.
func ValidateUsername(username string) error {
	return validateIdentifier(username, ErrUsernameRequired, ErrUsernameInvalid, ErrUsernameLength)
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

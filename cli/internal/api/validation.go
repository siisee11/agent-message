package api

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{3,32}$`)

func validateServerURLInput(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("server URL is required")
	}
	if hasControlCharacters(trimmed) {
		return "", errors.New("server URL must not contain control characters")
	}
	return trimmed, nil
}

func validateUsername(username string) (string, error) {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return "", errors.New("username is required")
	}
	if hasControlCharacters(trimmed) {
		return "", errors.New("username must not contain control characters")
	}
	if !usernamePattern.MatchString(trimmed) {
		return "", errors.New("username must be 3-32 chars and use only letters, numbers, dot, underscore, or hyphen")
	}
	return trimmed, nil
}

func validateResourceID(label, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	if hasControlCharacters(trimmed) {
		return "", fmt.Errorf("%s must not contain control characters", label)
	}
	if strings.Contains(trimmed, "..") {
		return "", fmt.Errorf("%s must not contain dot segments", label)
	}
	if strings.Contains(trimmed, "%") {
		return "", fmt.Errorf("%s must not contain percent-encoded segments", label)
	}
	if strings.ContainsAny(trimmed, `/\?#`) {
		return "", fmt.Errorf("%s must not contain path or query syntax", label)
	}
	return trimmed, nil
}

func validateWatcherSessionID(value string) (string, error) {
	return validateResourceID("watcher session id", value)
}

func validateAttachmentPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("attachment path is required")
	}
	if hasControlCharacters(trimmed) {
		return "", errors.New("attachment path must not contain control characters")
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == "." || cleaned == "" {
		return "", errors.New("attachment path must point to a file")
	}
	return trimmed, nil
}

func hasControlCharacters(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

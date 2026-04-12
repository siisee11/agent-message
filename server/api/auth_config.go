package api

import (
	"net/http"
	"strings"
	"time"
)

const (
	defaultSessionTTL        = 30 * 24 * time.Hour
	defaultSessionCookieName = "agent_message_session"
)

type AuthConfig struct {
	SessionTTL     time.Duration
	SessionCookie  string
	AllowedOrigins []string
}

func normalizeAuthConfig(cfg AuthConfig, allowedOrigins []string) AuthConfig {
	sessionTTL := cfg.SessionTTL
	if sessionTTL <= 0 {
		sessionTTL = defaultSessionTTL
	}

	sessionCookie := strings.TrimSpace(cfg.SessionCookie)
	if sessionCookie == "" {
		sessionCookie = defaultSessionCookieName
	}

	origins := allowedOrigins
	if len(cfg.AllowedOrigins) > 0 {
		origins = cfg.AllowedOrigins
	}

	return AuthConfig{
		SessionTTL:     sessionTTL,
		SessionCookie:  sessionCookie,
		AllowedOrigins: normalizeAllowedOrigins(origins),
	}
}

func sessionCookieSameSite() http.SameSite {
	return http.SameSiteStrictMode
}

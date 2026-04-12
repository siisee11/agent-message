package api

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agent-message/server/models"
	"agent-message/server/store"
)

var (
	errAuthTokenRequired = errors.New("missing auth token")
	errAuthTokenInvalid  = errors.New("invalid auth token")
)

type authSource string

const (
	authSourceBearer authSource = "bearer"
	authSourceCookie authSource = "cookie"
	authSourceQuery  authSource = "query"
)

type authenticator struct {
	store          store.Store
	sessionTTL     time.Duration
	sessionCookie  string
	allowedOrigins []string
	nowFn          func() time.Time
}

func newAuthenticator(s store.Store, cfg AuthConfig) *authenticator {
	normalized := normalizeAuthConfig(cfg, cfg.AllowedOrigins)
	return &authenticator{
		store:          s,
		sessionTTL:     normalized.SessionTTL,
		sessionCookie:  normalized.SessionCookie,
		allowedOrigins: normalized.AllowedOrigins,
		nowFn:          time.Now,
	}
}

func (a *authenticator) authenticateRequest(r *http.Request, allowQueryToken bool) (models.User, string, authSource, error) {
	token, source, err := a.resolveSessionToken(r, allowQueryToken)
	if err != nil {
		return models.User{}, "", "", err
	}

	session, err := a.store.GetSessionByToken(r.Context(), token)
	if err != nil {
		return models.User{}, "", "", err
	}

	if a.sessionExpired(session) {
		_ = a.store.DeleteSessionByToken(r.Context(), token)
		return models.User{}, "", "", store.ErrNotFound
	}

	user, err := a.store.GetUserByID(r.Context(), session.UserID)
	if err != nil {
		return models.User{}, "", "", err
	}

	return user, token, source, nil
}

func (a *authenticator) sessionExpired(session models.Session) bool {
	now := a.nowFn().UTC()
	if !session.ExpiresAt.IsZero() {
		return !session.ExpiresAt.After(now)
	}
	return !session.CreatedAt.Add(a.sessionTTL).After(now)
}

func (a *authenticator) validateStateChangingCookieRequest(r *http.Request, source authSource) error {
	if source != authSourceCookie {
		return nil
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return nil
	}

	rawOrigin := strings.TrimSpace(r.Header.Get("Origin"))
	if rawOrigin == "" {
		switch strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))) {
		case "same-origin", "same-site", "none":
			return nil
		default:
			return errors.New("missing origin")
		}
	}

	origin, err := url.Parse(rawOrigin)
	if err != nil || origin.Scheme == "" || origin.Host == "" {
		return errors.New("invalid origin")
	}

	if requestOrigin := requestOrigin(r); requestOrigin != "" && strings.EqualFold(rawOrigin, requestOrigin) {
		return nil
	}

	for _, allowedOrigin := range a.allowedOrigins {
		if allowedOrigin == "*" {
			continue
		}
		if strings.EqualFold(rawOrigin, allowedOrigin) {
			return nil
		}
	}

	return errors.New("invalid origin")
}

func (a *authenticator) resolveSessionToken(r *http.Request, allowQueryToken bool) (string, authSource, error) {
	if headerValue := strings.TrimSpace(r.Header.Get("Authorization")); headerValue != "" {
		token, err := parseBearerToken(headerValue)
		if err != nil {
			return "", "", errAuthTokenInvalid
		}
		return token, authSourceBearer, nil
	}

	if allowQueryToken {
		if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
			return token, authSourceQuery, nil
		}
	}

	cookie, err := r.Cookie(a.sessionCookie)
	if err == nil {
		token := strings.TrimSpace(cookie.Value)
		if token != "" {
			return token, authSourceCookie, nil
		}
	}

	return "", "", errAuthTokenRequired
}

func requestOrigin(r *http.Request) string {
	scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	if comma := strings.IndexByte(scheme, ','); comma >= 0 {
		scheme = strings.TrimSpace(scheme[:comma])
	}

	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if comma := strings.IndexByte(host, ','); comma >= 0 {
		host = strings.TrimSpace(host[:comma])
	}
	if scheme == "" || host == "" {
		return ""
	}
	return scheme + "://" + host
}

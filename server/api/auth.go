package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-message/server/models"
	"agent-message/server/store"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const sessionTokenByteLen = 32

type authHandler struct {
	store    store.Store
	config   AuthConfig
	nowFn    func() time.Time
	tokenFn  func() (string, error)
	bcryptFn func(password string) (string, error)
}

func newAuthHandler(s store.Store) *authHandler {
	return &authHandler{
		store:    s,
		config:   normalizeAuthConfig(AuthConfig{}, nil),
		nowFn:    time.Now,
		tokenFn:  generateSessionToken,
		bcryptFn: hashPassword,
	}
}

func (h *authHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req models.RegisterRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	accountID := req.AccountIDValue()
	if _, err := h.store.GetUserByAccountID(r.Context(), accountID); err == nil {
		writeError(w, http.StatusConflict, "account_id already exists")
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	desiredUsername := req.UsernameValue()
	if _, err := h.store.GetUserByUsername(r.Context(), desiredUsername); err == nil {
		writeError(w, http.StatusConflict, "username already exists")
		return
	} else if !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	password := req.Password
	if password == "" {
		password = req.LegacyPIN
	}
	passwordHash, err := h.bcryptFn(password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	now := h.nowFn().UTC()
	user, err := h.store.CreateUser(r.Context(), models.CreateUserParams{
		ID:           uuid.NewString(),
		AccountID:    accountID,
		Username:     desiredUsername,
		PasswordHash: passwordHash,
		CreatedAt:    now,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	token, err := h.issueSession(r, user.ID, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	h.writeSessionCookie(w, r, token, now.Add(h.config.SessionTTL))
	writeJSON(w, http.StatusCreated, models.AuthResponse{
		Token: token,
		User:  user.Profile(),
	})
}

func (h *authHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req models.LoginRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.store.GetUserByAccountID(r.Context(), req.AccountIDValue())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	password := req.Password
	if password == "" {
		password = req.LegacyPIN
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	now := h.nowFn().UTC()
	token, err := h.issueSession(r, user.ID, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	h.writeSessionCookie(w, r, token, now.Add(h.config.SessionTTL))
	writeJSON(w, http.StatusOK, models.AuthResponse{
		Token: token,
		User:  user.Profile(),
	})
}

func (h *authHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodNotAllowed(w, http.MethodDelete)
		return
	}

	token, ok := tokenFromContext(r.Context())
	if !ok || token == "" {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	if err := h.store.DeleteSessionByToken(r.Context(), token); err != nil && !errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "failed to logout")
		return
	}

	h.clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *authHandler) issueSession(r *http.Request, userID string, now time.Time) (string, error) {
	token, err := h.tokenFn()
	if err != nil {
		return "", err
	}

	_, err = h.store.CreateSession(r.Context(), models.CreateSessionParams{
		Token:     token,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(h.config.SessionTTL),
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

func (h *authHandler) writeSessionCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.config.SessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   requestIsSecure(r),
		SameSite: sessionCookieSameSite(),
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
	w.Header().Set("Cache-Control", "no-store")
}

func (h *authHandler) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.config.SessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   requestIsSecure(r),
		SameSite: sessionCookieSameSite(),
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	})
	w.Header().Set("Cache-Control", "no-store")
}

func requestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}

	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		return false
	}
	if comma := strings.IndexByte(proto, ','); comma >= 0 {
		proto = strings.TrimSpace(proto[:comma])
	}
	return strings.EqualFold(proto, "https")
}

func decodeJSONBody(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}

func parseBearerToken(headerValue string) (string, error) {
	parts := strings.SplitN(strings.TrimSpace(headerValue), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization header")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("empty bearer token")
	}
	return token, nil
}

func generateSessionToken() (string, error) {
	raw := make([]byte, sessionTokenByteLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

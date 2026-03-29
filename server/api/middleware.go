package api

import (
	"errors"
	"net/http"
	"slices"
	"strings"

	"agent-messenger/server/store"
)

func BearerAuthMiddleware(dataStore store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := parseBearerToken(r.Header.Get("Authorization"))
			if err != nil {
				writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
				return
			}

			user, err := dataStore.GetUserBySessionToken(r.Context(), token)
			if err != nil {
				if errors.Is(err, store.ErrNotFound) {
					writeError(w, http.StatusUnauthorized, "invalid session token")
					return
				}
				writeError(w, http.StatusInternalServerError, "failed to validate bearer token")
				return
			}

			next.ServeHTTP(w, r.WithContext(contextWithAuth(r.Context(), user, token)))
		})
	}
}

func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	origins := normalizeAllowedOrigins(allowedOrigins)
	allowAny := slices.Contains(origins, "*")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				w.Header().Add("Vary", "Origin")
				w.Header().Add("Vary", "Access-Control-Request-Method")
				w.Header().Add("Vary", "Access-Control-Request-Headers")
			}

			if allowAny {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" && slices.Contains(origins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func normalizeAllowedOrigins(origins []string) []string {
	if len(origins) == 0 {
		return []string{"*"}
	}

	out := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

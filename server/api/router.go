package api

import (
	"encoding/json"
	"net/http"

	"agent-messenger/server/store"
	"agent-messenger/server/ws"
)

type Dependencies struct {
	Store              store.Store
	Hub                *ws.Hub
	CORSAllowedOrigins []string
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)

	authHandler := newAuthHandler(deps.Store)
	mux.HandleFunc("/api/auth/register", authHandler.handleRegister)
	mux.HandleFunc("/api/auth/login", authHandler.handleLogin)
	mux.Handle("/api/auth/logout", BearerAuthMiddleware(deps.Store)(http.HandlerFunc(authHandler.handleLogout)))

	return CORSMiddleware(deps.CORSAllowedOrigins)(mux)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

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
	usersHandler := newUsersHandler(deps.Store)
	conversationsHandler := newConversationsHandler(deps.Store)
	authRequired := BearerAuthMiddleware(deps.Store)

	mux.HandleFunc("/api/auth/register", authHandler.handleRegister)
	mux.HandleFunc("/api/auth/login", authHandler.handleLogin)
	mux.Handle("/api/auth/logout", authRequired(http.HandlerFunc(authHandler.handleLogout)))

	mux.Handle("/api/users", authRequired(http.HandlerFunc(usersHandler.handleUsers)))
	mux.Handle("/api/users/me", authRequired(http.HandlerFunc(usersHandler.handleMe)))

	mux.Handle("/api/conversations", authRequired(http.HandlerFunc(conversationsHandler.handleConversationsCollection)))
	mux.Handle("/api/conversations/", authRequired(http.HandlerFunc(conversationsHandler.handleConversationDetail)))

	return CORSMiddleware(deps.CORSAllowedOrigins)(mux)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

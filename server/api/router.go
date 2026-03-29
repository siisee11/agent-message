package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"agent-messenger/server/store"
	"agent-messenger/server/ws"
)

type Dependencies struct {
	Store              store.Store
	Hub                *ws.Hub
	CORSAllowedOrigins []string
	UploadDir          string
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)

	uploadDir := strings.TrimSpace(deps.UploadDir)
	if uploadDir == "" {
		uploadDir = defaultUploadDir
	}

	hub := deps.Hub
	if hub == nil {
		hub = ws.NewHub()
	}

	authHandler := newAuthHandler(deps.Store)
	usersHandler := newUsersHandler(deps.Store)
	conversationsHandler := newConversationsHandler(deps.Store)
	messagesHandler := newMessagesHandler(deps.Store, hub)
	websocketHandler := newWebSocketHandler(deps.Store, hub)
	messagesHandler.uploadDir = uploadDir
	uploadHandler := newUploadHandler(uploadDir)
	authRequired := BearerAuthMiddleware(deps.Store)

	mux.HandleFunc("/api/auth/register", authHandler.handleRegister)
	mux.HandleFunc("/api/auth/login", authHandler.handleLogin)
	mux.Handle("/api/auth/logout", authRequired(http.HandlerFunc(authHandler.handleLogout)))

	mux.Handle("/api/users", authRequired(http.HandlerFunc(usersHandler.handleUsers)))
	mux.Handle("/api/users/me", authRequired(http.HandlerFunc(usersHandler.handleMe)))

	mux.Handle("/api/conversations", authRequired(http.HandlerFunc(conversationsHandler.handleConversationsCollection)))
	mux.Handle("/api/conversations/", authRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, isMessagesPath := conversationMessagesPath(r.URL.Path); isMessagesPath {
			messagesHandler.handleConversationMessages(w, r)
			return
		}
		conversationsHandler.handleConversationDetail(w, r)
	})))

	mux.Handle("/api/messages/", authRequired(http.HandlerFunc(messagesHandler.handleMessageByID)))
	mux.Handle("/api/upload", authRequired(http.HandlerFunc(uploadHandler.handleUpload)))
	mux.Handle("/static/uploads/", http.StripPrefix(staticUploadsPrefix, http.FileServer(http.Dir(uploadDir))))
	mux.HandleFunc("/ws", websocketHandler.handleWebSocket)

	return CORSMiddleware(deps.CORSAllowedOrigins)(mux)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

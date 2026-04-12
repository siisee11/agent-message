package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"agent-message/server/push"
	"agent-message/server/realtime"
	"agent-message/server/store"
)

type Dependencies struct {
	Store              store.Store
	Hub                *realtime.Hub
	WatcherPresence    *realtime.WatcherPresence
	Push               *push.Service
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
		hub = realtime.NewHub()
	}
	watcherPresence := deps.WatcherPresence
	if watcherPresence == nil {
		watcherPresence = realtime.NewWatcherPresence(realtime.DefaultWatcherPresenceTTL)
	}

	authHandler := newAuthHandler(deps.Store)
	catalogHandler := newCatalogHandler()
	usersHandler := newUsersHandler(deps.Store)
	conversationsHandler := newConversationsHandler(deps.Store, hub, watcherPresence)
	messagesHandler := newMessagesHandler(deps.Store, hub, watcherPresence)
	messagesHandler.notifier = deps.Push
	eventStreamHandler := newEventStreamHandler(deps.Store, hub, watcherPresence)
	watcherPresenceHandler := newWatcherPresenceHandler(watcherPresence, hub)
	pushHandler := newPushHandler(deps.Store, deps.Push)
	messagesHandler.uploadDir = uploadDir
	uploadHandler := newUploadHandler(uploadDir)
	authRequired := BearerAuthMiddleware(deps.Store)

	mux.HandleFunc("/api/auth/register", authHandler.handleRegister)
	mux.HandleFunc("/api/auth/login", authHandler.handleLogin)
	mux.HandleFunc("/api/catalog/prompt", catalogHandler.handlePrompt)
	mux.Handle("/api/auth/logout", authRequired(http.HandlerFunc(authHandler.handleLogout)))

	mux.Handle("/api/users", authRequired(http.HandlerFunc(usersHandler.handleUsers)))
	mux.Handle("/api/users/me", authRequired(http.HandlerFunc(usersHandler.handleMe)))
	mux.Handle("/api/push/config", authRequired(http.HandlerFunc(pushHandler.handleConfig)))
	mux.Handle("/api/push/subscriptions", authRequired(http.HandlerFunc(pushHandler.handleSubscriptions)))
	mux.Handle("/api/watchers/heartbeat", authRequired(http.HandlerFunc(watcherPresenceHandler.handleHeartbeat)))
	mux.Handle("/api/watchers/sessions/", authRequired(http.HandlerFunc(watcherPresenceHandler.handleSessionByID)))

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
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not found")
	})
	mux.HandleFunc("/api/events", eventStreamHandler.handleEventStream)
	mux.Handle("/static/uploads/", http.StripPrefix(staticUploadsPrefix, http.FileServer(http.Dir(uploadDir))))

	return CORSMiddleware(deps.CORSAllowedOrigins)(mux)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

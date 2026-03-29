package api

import (
	"encoding/json"
	"net/http"

	"agent-messenger/server/store"
	"agent-messenger/server/ws"
)

type Dependencies struct {
	Store store.Store
	Hub   *ws.Hub
}

func NewRouter(_ Dependencies) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	return mux
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

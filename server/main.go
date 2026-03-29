package main

import (
	"log"
	"net/http"

	"agent-messenger/server/api"
	"agent-messenger/server/store"
	"agent-messenger/server/ws"
)

func main() {
	hub := ws.NewHub()
	store := store.NewNoopStore()

	handler := api.NewRouter(api.Dependencies{
		Store: store,
		Hub:   hub,
	})

	addr := ":8080"
	log.Printf("agent-messenger server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

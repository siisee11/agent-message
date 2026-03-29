package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"agent-messenger/server/api"
	"agent-messenger/server/store"
	"agent-messenger/server/ws"
)

const (
	defaultServerAddr = ":8080"
	defaultSQLiteDSN  = "./agent_messenger.sqlite"
)

type config struct {
	ServerAddr         string
	SQLiteDSN          string
	CORSAllowedOrigins []string
}

func main() {
	cfg := loadConfigFromEnv()

	ctx := context.Background()
	hub := ws.NewHub()
	dataStore, err := store.NewSQLiteStore(ctx, cfg.SQLiteDSN)
	if err != nil {
		log.Fatalf("failed to initialize sqlite store: %v", err)
	}
	defer func() {
		if err := dataStore.Close(); err != nil {
			log.Printf("failed to close store: %v", err)
		}
	}()

	handler := api.NewRouter(api.Dependencies{
		Store:              dataStore,
		Hub:                hub,
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
	})

	log.Printf("agent-messenger server listening on %s", cfg.ServerAddr)
	log.Printf("sqlite dsn: %s", cfg.SQLiteDSN)
	log.Printf("cors allowed origins: %s", strings.Join(cfg.CORSAllowedOrigins, ","))
	if err := http.ListenAndServe(cfg.ServerAddr, handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

func loadConfigFromEnv() config {
	return config{
		ServerAddr:         envOrDefault("SERVER_ADDR", defaultServerAddr),
		SQLiteDSN:          envOrDefault("SQLITE_DSN", defaultSQLiteDSN),
		CORSAllowedOrigins: parseCSVEnv("CORS_ALLOWED_ORIGINS", "*"),
	}
}

func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func parseCSVEnv(name, fallback string) []string {
	raw := envOrDefault(name, fallback)
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
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

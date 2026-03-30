package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"agent-messenger/server/api"
	"agent-messenger/server/realtime"
	"agent-messenger/server/store"
)

const (
	defaultServerAddr = ":8080"
	defaultDBDriver   = "sqlite"
	defaultSQLiteDSN  = "./agent_messenger.sqlite"
	defaultUploadDir  = "./uploads"
)

type config struct {
	ServerAddr         string
	DBDriver           string
	SQLiteDSN          string
	PostgresDSN        string
	CORSAllowedOrigins []string
	UploadDir          string
}

func main() {
	cfg := loadConfigFromEnv()

	ctx := context.Background()
	hub := realtime.NewHub()
	dataStore, driver, err := openStore(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}
	defer func() {
		if err := dataStore.Close(); err != nil {
			log.Printf("failed to close store: %v", err)
		}
	}()

	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("failed to create upload dir: %v", err)
	}

	handler := api.NewRouter(api.Dependencies{
		Store:              dataStore,
		Hub:                hub,
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		UploadDir:          cfg.UploadDir,
	})

	log.Printf("agent-messenger server listening on %s", cfg.ServerAddr)
	log.Printf("db driver: %s", driver)
	if driver == defaultDBDriver {
		log.Printf("sqlite dsn: %s", cfg.SQLiteDSN)
	}
	log.Printf("cors allowed origins: %s", strings.Join(cfg.CORSAllowedOrigins, ","))
	log.Printf("upload dir: %s", cfg.UploadDir)
	if err := http.ListenAndServe(cfg.ServerAddr, handler); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

func loadConfigFromEnv() config {
	return config{
		ServerAddr:         envOrDefault("SERVER_ADDR", defaultServerAddr),
		DBDriver:           envOrDefault("DB_DRIVER", defaultDBDriver),
		SQLiteDSN:          envOrDefault("SQLITE_DSN", defaultSQLiteDSN),
		PostgresDSN:        envOrDefault("POSTGRES_DSN", strings.TrimSpace(os.Getenv("DATABASE_URL"))),
		CORSAllowedOrigins: parseCSVEnv("CORS_ALLOWED_ORIGINS", "*"),
		UploadDir:          envOrDefault("UPLOAD_DIR", defaultUploadDir),
	}
}

func openStore(ctx context.Context, cfg config) (store.Store, string, error) {
	driver := normalizeDBDriver(cfg.DBDriver)
	switch driver {
	case "sqlite":
		dataStore, err := store.NewSQLiteStore(ctx, cfg.SQLiteDSN)
		if err != nil {
			return nil, "", err
		}
		return dataStore, driver, nil
	case "postgres":
		dataStore, err := store.NewPostgresStore(ctx, cfg.PostgresDSN)
		if err != nil {
			return nil, "", err
		}
		return dataStore, driver, nil
	default:
		return nil, "", errors.New("unsupported DB_DRIVER: " + driver)
	}
}

func normalizeDBDriver(raw string) string {
	driver := strings.ToLower(strings.TrimSpace(raw))
	if driver == "" {
		return defaultDBDriver
	}
	return driver
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

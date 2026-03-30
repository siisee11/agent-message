package main

import (
	"context"
	"path/filepath"
	"testing"
)

func TestParseCSVEnv(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example.com, https://b.example.com")
	got := parseCSVEnv("CORS_ALLOWED_ORIGINS", "*")
	if len(got) != 2 || got[0] != "https://a.example.com" || got[1] != "https://b.example.com" {
		t.Fatalf("unexpected parseCSVEnv output: %#v", got)
	}
}

func TestParseCSVEnvDefaults(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "   ")
	got := parseCSVEnv("CORS_ALLOWED_ORIGINS", "*")
	if len(got) != 1 || got[0] != "*" {
		t.Fatalf("unexpected default output: %#v", got)
	}
}

func TestLoadConfigFromEnvUploadDir(t *testing.T) {
	t.Setenv("UPLOAD_DIR", "/tmp/uploads-test")
	cfg := loadConfigFromEnv()
	if cfg.UploadDir != "/tmp/uploads-test" {
		t.Fatalf("expected upload dir /tmp/uploads-test, got %q", cfg.UploadDir)
	}
}

func TestNormalizeDBDriver(t *testing.T) {
	if got := normalizeDBDriver(""); got != "sqlite" {
		t.Fatalf("expected sqlite default, got %q", got)
	}
	if got := normalizeDBDriver("  POSTGRES "); got != "postgres" {
		t.Fatalf("expected postgres normalization, got %q", got)
	}
}

func TestOpenStoreSQLite(t *testing.T) {
	cfg := config{
		DBDriver:  "sqlite",
		SQLiteDSN: filepath.Join(t.TempDir(), "main-open-store.sqlite"),
	}
	dataStore, driver, err := openStore(context.Background(), cfg)
	if err != nil {
		t.Fatalf("openStore sqlite failed: %v", err)
	}
	t.Cleanup(func() {
		_ = dataStore.Close()
	})

	if driver != "sqlite" {
		t.Fatalf("expected sqlite driver, got %q", driver)
	}
}

func TestOpenStorePostgresRequiresDSN(t *testing.T) {
	cfg := config{DBDriver: "postgres"}
	_, _, err := openStore(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for missing postgres dsn")
	}
}

func TestOpenStoreRejectsUnknownDriver(t *testing.T) {
	cfg := config{DBDriver: "mystery"}
	_, _, err := openStore(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected unsupported driver error")
	}
}

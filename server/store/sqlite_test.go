package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"agent-messenger/server/models"
)

func TestSQLiteStoreAppliesMigrations(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")

	s, err := NewSQLiteStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count schema migrations: %v", err)
	}

	if count != len(sqliteMigrations) {
		t.Fatalf("expected %d applied migrations, got %d", len(sqliteMigrations), count)
	}
}

func TestSQLiteStoreUserAndSessionAuthFlow(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "auth.sqlite")

	s, err := NewSQLiteStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	now := time.Now().UTC().Truncate(time.Microsecond)

	user, err := s.CreateUser(ctx, models.CreateUserParams{
		ID:        "user-1",
		Username:  "alice",
		PINHash:   "bcrypt-hash",
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	if user.ID != "user-1" || user.Username != "alice" {
		t.Fatalf("unexpected user: %+v", user)
	}

	gotByUsername, err := s.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if gotByUsername.ID != user.ID {
		t.Fatalf("expected same user by username, got %+v", gotByUsername)
	}

	session, err := s.CreateSession(ctx, models.CreateSessionParams{
		Token:     "token-1",
		UserID:    user.ID,
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session.Token != "token-1" || session.UserID != user.ID {
		t.Fatalf("unexpected session: %+v", session)
	}

	resolvedUser, err := s.GetUserBySessionToken(ctx, "token-1")
	if err != nil {
		t.Fatalf("GetUserBySessionToken() error = %v", err)
	}
	if resolvedUser.ID != user.ID {
		t.Fatalf("expected resolved user %q, got %q", user.ID, resolvedUser.ID)
	}

	if err := s.DeleteSessionByToken(ctx, "token-1"); err != nil {
		t.Fatalf("DeleteSessionByToken() error = %v", err)
	}

	_, err = s.GetSessionByToken(ctx, "token-1")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

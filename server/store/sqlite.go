package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-messenger/server/models"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(ctx context.Context, dsn string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	s := &SQLiteStore{db: db}

	if err := s.configure(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) CreateUser(ctx context.Context, params models.CreateUserParams) (models.User, error) {
	const query = `
		INSERT INTO users (id, username, pin_hash, created_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query, params.ID, params.Username, params.PINHash, formatTime(params.CreatedAt))
	if err != nil {
		return models.User{}, fmt.Errorf("insert user: %w", err)
	}
	return s.GetUserByID(ctx, params.ID)
}

func (s *SQLiteStore) GetUserByUsername(ctx context.Context, username string) (models.User, error) {
	const query = `
		SELECT id, username, pin_hash, created_at
		FROM users
		WHERE username = ?
	`

	return s.getUserByQuery(ctx, query, username)
}

func (s *SQLiteStore) GetUserByID(ctx context.Context, userID string) (models.User, error) {
	const query = `
		SELECT id, username, pin_hash, created_at
		FROM users
		WHERE id = ?
	`

	return s.getUserByQuery(ctx, query, userID)
}

func (s *SQLiteStore) CreateSession(ctx context.Context, params models.CreateSessionParams) (models.Session, error) {
	const query = `
		INSERT INTO sessions (token, user_id, created_at)
		VALUES (?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query, params.Token, params.UserID, formatTime(params.CreatedAt))
	if err != nil {
		return models.Session{}, fmt.Errorf("insert session: %w", err)
	}

	return s.GetSessionByToken(ctx, params.Token)
}

func (s *SQLiteStore) GetSessionByToken(ctx context.Context, token string) (models.Session, error) {
	const query = `
		SELECT token, user_id, created_at
		FROM sessions
		WHERE token = ?
	`

	row := s.db.QueryRowContext(ctx, query, token)

	var session models.Session
	var createdAtText string
	if err := row.Scan(&session.Token, &session.UserID, &createdAtText); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Session{}, ErrNotFound
		}
		return models.Session{}, fmt.Errorf("select session by token: %w", err)
	}

	createdAt, err := parseStoredTime(createdAtText)
	if err != nil {
		return models.Session{}, fmt.Errorf("parse session created_at: %w", err)
	}
	session.CreatedAt = createdAt

	return session, nil
}

func (s *SQLiteStore) DeleteSessionByToken(ctx context.Context, token string) error {
	const query = `DELETE FROM sessions WHERE token = ?`
	res, err := s.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete session rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *SQLiteStore) GetUserBySessionToken(ctx context.Context, token string) (models.User, error) {
	const query = `
		SELECT u.id, u.username, u.pin_hash, u.created_at
		FROM users u
		INNER JOIN sessions s ON s.user_id = u.id
		WHERE s.token = ?
	`

	return s.getUserByQuery(ctx, query, token)
}

func (s *SQLiteStore) SearchUsersByUsername(_ context.Context, _ models.SearchUsersParams) ([]models.User, error) {
	return nil, ErrNotImplemented
}

func (s *SQLiteStore) ListConversationsByUser(_ context.Context, _ models.ListUserConversationsParams) ([]models.ConversationSummary, error) {
	return nil, ErrNotImplemented
}

func (s *SQLiteStore) GetOrCreateDirectConversation(_ context.Context, _ models.GetOrCreateDirectConversationParams) (models.Conversation, error) {
	return models.Conversation{}, ErrNotImplemented
}

func (s *SQLiteStore) GetConversationByIDForUser(_ context.Context, _ models.GetConversationForUserParams) (models.ConversationDetails, error) {
	return models.ConversationDetails{}, ErrNotImplemented
}

func (s *SQLiteStore) ListMessagesByConversation(_ context.Context, _ models.ListConversationMessagesParams) ([]models.MessageDetails, error) {
	return nil, ErrNotImplemented
}

func (s *SQLiteStore) CreateMessage(_ context.Context, _ models.CreateMessageParams) (models.Message, error) {
	return models.Message{}, ErrNotImplemented
}

func (s *SQLiteStore) UpdateMessage(_ context.Context, _ models.UpdateMessageParams) (models.Message, error) {
	return models.Message{}, ErrNotImplemented
}

func (s *SQLiteStore) SoftDeleteMessage(_ context.Context, _ models.SoftDeleteMessageParams) (models.Message, error) {
	return models.Message{}, ErrNotImplemented
}

func (s *SQLiteStore) configure(ctx context.Context) error {
	const query = `PRAGMA foreign_keys = ON`
	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("set foreign_keys pragma: %w", err)
	}
	return nil
}

func (s *SQLiteStore) getUserByQuery(ctx context.Context, query string, arg string) (models.User, error) {
	row := s.db.QueryRowContext(ctx, query, arg)

	var user models.User
	var createdAtText string
	if err := row.Scan(&user.ID, &user.Username, &user.PINHash, &createdAtText); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, ErrNotFound
		}
		return models.User{}, fmt.Errorf("scan user: %w", err)
	}

	createdAt, err := parseStoredTime(createdAtText)
	if err != nil {
		return models.User{}, fmt.Errorf("parse user created_at: %w", err)
	}
	user.CreatedAt = createdAt

	return user, nil
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseStoredTime(v string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(v))
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

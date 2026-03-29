package store

import (
	"context"
	"database/sql"
	"fmt"
)

type migration struct {
	version int
	name    string
	sql     string
}

var sqliteMigrations = []migration{
	{
		version: 1,
		name:    "create_users",
		sql: `
			CREATE TABLE users (
				id TEXT PRIMARY KEY,
				username TEXT NOT NULL UNIQUE,
				pin_hash TEXT NOT NULL,
				created_at TEXT NOT NULL
			);
		`,
	},
	{
		version: 2,
		name:    "create_conversations",
		sql: `
			CREATE TABLE conversations (
				id TEXT PRIMARY KEY,
				participant_a TEXT NOT NULL,
				participant_b TEXT NOT NULL,
				created_at TEXT NOT NULL,
				FOREIGN KEY(participant_a) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY(participant_b) REFERENCES users(id) ON DELETE CASCADE,
				CHECK(participant_a <> participant_b)
			);
		`,
	},
	{
		version: 3,
		name:    "create_messages",
		sql: `
			CREATE TABLE messages (
				id TEXT PRIMARY KEY,
				conversation_id TEXT NOT NULL,
				sender_id TEXT NOT NULL,
				content TEXT NULL,
				attachment_url TEXT NULL,
				attachment_type TEXT NULL CHECK(attachment_type IN ('image', 'file') OR attachment_type IS NULL),
				edited INTEGER NOT NULL DEFAULT 0,
				deleted INTEGER NOT NULL DEFAULT 0,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				FOREIGN KEY(conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
				FOREIGN KEY(sender_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`,
	},
	{
		version: 4,
		name:    "create_reactions",
		sql: `
			CREATE TABLE reactions (
				id TEXT PRIMARY KEY,
				message_id TEXT NOT NULL,
				user_id TEXT NOT NULL,
				emoji TEXT NOT NULL,
				created_at TEXT NOT NULL,
				FOREIGN KEY(message_id) REFERENCES messages(id) ON DELETE CASCADE,
				FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
				UNIQUE(message_id, user_id, emoji)
			);
		`,
	},
	{
		version: 5,
		name:    "create_sessions",
		sql: `
			CREATE TABLE sessions (
				token TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				created_at TEXT NOT NULL,
				FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`,
	},
	{
		version: 6,
		name:    "create_indexes",
		sql: `
			CREATE INDEX idx_messages_conversation_id_created_at ON messages(conversation_id, created_at);
			CREATE INDEX idx_reactions_message_id ON reactions(message_id);
			CREATE INDEX idx_sessions_user_id ON sessions(user_id);
		`,
	},
}

func (s *SQLiteStore) migrate(ctx context.Context) error {
	if err := s.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	applied, err := s.appliedMigrationSet(ctx)
	if err != nil {
		return err
	}

	for _, migration := range sqliteMigrations {
		if applied[migration.version] {
			continue
		}
		if err := s.applyMigration(ctx, migration); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLiteStore) ensureMigrationsTable(ctx context.Context) error {
	const query = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		);
	`
	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}
	return nil
}

func (s *SQLiteStore) appliedMigrationSet(ctx context.Context) (map[int]bool, error) {
	const query = `SELECT version FROM schema_migrations`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", err)
	}

	return applied, nil
}

func (s *SQLiteStore) applyMigration(ctx context.Context, migration migration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, migration.sql); err != nil {
		return fmt.Errorf("apply migration %d (%s): %w", migration.version, migration.name, err)
	}

	if err := insertMigrationRecord(ctx, tx, migration.version, migration.name); err != nil {
		return fmt.Errorf("record migration %d (%s): %w", migration.version, migration.name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %d (%s): %w", migration.version, migration.name, err)
	}
	return nil
}

func insertMigrationRecord(ctx context.Context, tx *sql.Tx, version int, name string) error {
	const query = `INSERT INTO schema_migrations (version, name) VALUES (?, ?)`
	_, err := tx.ExecContext(ctx, query, version, name)
	return err
}

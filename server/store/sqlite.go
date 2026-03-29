package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
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

func (s *SQLiteStore) SearchUsersByUsername(ctx context.Context, params models.SearchUsersParams) ([]models.User, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	like := strings.TrimSpace(params.Query) + "%"
	const query = `
		SELECT id, username, pin_hash, created_at
		FROM users
		WHERE username LIKE ? COLLATE NOCASE
		ORDER BY username ASC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, like, limit)
	if err != nil {
		return nil, fmt.Errorf("search users by username: %w", err)
	}
	defer rows.Close()

	users := make([]models.User, 0, limit)
	for rows.Next() {
		var user models.User
		var createdAtText string
		if err := rows.Scan(&user.ID, &user.Username, &user.PINHash, &createdAtText); err != nil {
			return nil, fmt.Errorf("scan searched user: %w", err)
		}
		createdAt, err := parseStoredTime(createdAtText)
		if err != nil {
			return nil, fmt.Errorf("parse searched user created_at: %w", err)
		}
		user.CreatedAt = createdAt
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate searched users: %w", err)
	}

	return users, nil
}

func (s *SQLiteStore) ListConversationsByUser(ctx context.Context, params models.ListUserConversationsParams) ([]models.ConversationSummary, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}

	const query = `
		SELECT
			c.id, c.participant_a, c.participant_b, c.created_at,
			ou.id, ou.username, ou.created_at,
			m.id, m.conversation_id, m.sender_id, m.content, m.attachment_url, m.attachment_type, m.edited, m.deleted, m.created_at, m.updated_at
		FROM conversations c
		INNER JOIN users ou ON ou.id = CASE WHEN c.participant_a = ? THEN c.participant_b ELSE c.participant_a END
		LEFT JOIN messages m ON m.id = (
			SELECT m2.id
			FROM messages m2
			WHERE m2.conversation_id = c.id
			ORDER BY m2.created_at DESC, m2.id DESC
			LIMIT 1
		)
		WHERE c.participant_a = ? OR c.participant_b = ?
		ORDER BY COALESCE(m.created_at, c.created_at) DESC, c.id DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, params.UserID, params.UserID, params.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("list conversations by user: %w", err)
	}
	defer rows.Close()

	summaries := make([]models.ConversationSummary, 0, limit)
	for rows.Next() {
		var (
			conversationCreatedAtText string
			otherUserCreatedAtText    string
			messageID                 sql.NullString
			messageConversationID     sql.NullString
			messageSenderID           sql.NullString
			messageContent            sql.NullString
			messageAttachmentURL      sql.NullString
			messageAttachmentType     sql.NullString
			messageEdited             sql.NullInt64
			messageDeleted            sql.NullInt64
			messageCreatedAtText      sql.NullString
			messageUpdatedAtText      sql.NullString
		)

		var summary models.ConversationSummary
		if err := rows.Scan(
			&summary.Conversation.ID,
			&summary.Conversation.ParticipantA,
			&summary.Conversation.ParticipantB,
			&conversationCreatedAtText,
			&summary.OtherUser.ID,
			&summary.OtherUser.Username,
			&otherUserCreatedAtText,
			&messageID,
			&messageConversationID,
			&messageSenderID,
			&messageContent,
			&messageAttachmentURL,
			&messageAttachmentType,
			&messageEdited,
			&messageDeleted,
			&messageCreatedAtText,
			&messageUpdatedAtText,
		); err != nil {
			return nil, fmt.Errorf("scan listed conversation: %w", err)
		}

		conversationCreatedAt, err := parseStoredTime(conversationCreatedAtText)
		if err != nil {
			return nil, fmt.Errorf("parse conversation created_at: %w", err)
		}
		summary.Conversation.CreatedAt = conversationCreatedAt

		otherUserCreatedAt, err := parseStoredTime(otherUserCreatedAtText)
		if err != nil {
			return nil, fmt.Errorf("parse other user created_at: %w", err)
		}
		summary.OtherUser.CreatedAt = otherUserCreatedAt

		if messageID.Valid {
			message, err := nullableMessageToModel(
				messageID,
				messageConversationID,
				messageSenderID,
				messageContent,
				messageAttachmentURL,
				messageAttachmentType,
				messageEdited,
				messageDeleted,
				messageCreatedAtText,
				messageUpdatedAtText,
			)
			if err != nil {
				return nil, fmt.Errorf("decode conversation last message: %w", err)
			}
			summary.LastMessage = &message
		}

		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate listed conversations: %w", err)
	}

	return summaries, nil
}

func (s *SQLiteStore) GetOrCreateDirectConversation(ctx context.Context, params models.GetOrCreateDirectConversationParams) (models.Conversation, error) {
	participants := []string{params.CurrentUserID, params.TargetUserID}
	slices.Sort(participants)
	participantA, participantB := participants[0], participants[1]
	if participantA == participantB {
		return models.Conversation{}, ErrForbidden
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Conversation{}, fmt.Errorf("begin get-or-create conversation tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	conversation, err := getConversationByParticipantsTx(ctx, tx, participantA, participantB)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return models.Conversation{}, fmt.Errorf("commit existing conversation tx: %w", err)
		}
		return conversation, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return models.Conversation{}, err
	}

	const insertQuery = `
		INSERT INTO conversations (id, participant_a, participant_b, created_at)
		VALUES (?, ?, ?, ?)
	`
	_, err = tx.ExecContext(
		ctx,
		insertQuery,
		params.ConversationID,
		participantA,
		participantB,
		formatTime(params.CreatedAt),
	)
	if err != nil {
		// A concurrent insert may have created this pair already.
		conversation, getErr := getConversationByParticipantsTx(ctx, tx, participantA, participantB)
		if getErr == nil {
			if err := tx.Commit(); err != nil {
				return models.Conversation{}, fmt.Errorf("commit raced conversation tx: %w", err)
			}
			return conversation, nil
		}
		return models.Conversation{}, fmt.Errorf("insert conversation: %w", err)
	}

	conversation, err = getConversationByIDTx(ctx, tx, params.ConversationID)
	if err != nil {
		return models.Conversation{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.Conversation{}, fmt.Errorf("commit created conversation tx: %w", err)
	}
	return conversation, nil
}

func (s *SQLiteStore) GetConversationByIDForUser(ctx context.Context, params models.GetConversationForUserParams) (models.ConversationDetails, error) {
	const query = `
		SELECT
			c.id, c.participant_a, c.participant_b, c.created_at,
			ua.id, ua.username, ua.created_at,
			ub.id, ub.username, ub.created_at
		FROM conversations c
		INNER JOIN users ua ON ua.id = c.participant_a
		INNER JOIN users ub ON ub.id = c.participant_b
		WHERE c.id = ?
	`

	row := s.db.QueryRowContext(ctx, query, params.ConversationID)

	var (
		details                   models.ConversationDetails
		conversationCreatedAtText string
		participantACreatedAtText string
		participantBCreatedAtText string
	)

	if err := row.Scan(
		&details.Conversation.ID,
		&details.Conversation.ParticipantA,
		&details.Conversation.ParticipantB,
		&conversationCreatedAtText,
		&details.ParticipantA.ID,
		&details.ParticipantA.Username,
		&participantACreatedAtText,
		&details.ParticipantB.ID,
		&details.ParticipantB.Username,
		&participantBCreatedAtText,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.ConversationDetails{}, ErrNotFound
		}
		return models.ConversationDetails{}, fmt.Errorf("select conversation details: %w", err)
	}

	if params.UserID != details.Conversation.ParticipantA && params.UserID != details.Conversation.ParticipantB {
		return models.ConversationDetails{}, ErrForbidden
	}

	conversationCreatedAt, err := parseStoredTime(conversationCreatedAtText)
	if err != nil {
		return models.ConversationDetails{}, fmt.Errorf("parse conversation details created_at: %w", err)
	}
	details.Conversation.CreatedAt = conversationCreatedAt

	participantACreatedAt, err := parseStoredTime(participantACreatedAtText)
	if err != nil {
		return models.ConversationDetails{}, fmt.Errorf("parse participant_a created_at: %w", err)
	}
	details.ParticipantA.CreatedAt = participantACreatedAt

	participantBCreatedAt, err := parseStoredTime(participantBCreatedAtText)
	if err != nil {
		return models.ConversationDetails{}, fmt.Errorf("parse participant_b created_at: %w", err)
	}
	details.ParticipantB.CreatedAt = participantBCreatedAt

	return details, nil
}

func (s *SQLiteStore) ListMessagesByConversation(ctx context.Context, params models.ListConversationMessagesParams) ([]models.MessageDetails, error) {
	if err := s.ensureConversationParticipant(ctx, params.ConversationID, params.UserID); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = models.DefaultMessagePageLimit
	}

	var (
		beforeCreatedAtText string
		beforeID            string
	)
	if params.BeforeMessageID != nil && strings.TrimSpace(*params.BeforeMessageID) != "" {
		beforeID = strings.TrimSpace(*params.BeforeMessageID)
		const beforeQuery = `
			SELECT created_at
			FROM messages
			WHERE id = ? AND conversation_id = ?
		`
		if err := s.db.QueryRowContext(ctx, beforeQuery, beforeID, params.ConversationID).Scan(&beforeCreatedAtText); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, fmt.Errorf("select before-cursor message: %w", err)
		}
	}

	query := `
		SELECT
			m.id, m.conversation_id, m.sender_id, m.content, m.attachment_url, m.attachment_type, m.edited, m.deleted, m.created_at, m.updated_at,
			u.id, u.username, u.created_at
		FROM messages m
		INNER JOIN users u ON u.id = m.sender_id
		WHERE m.conversation_id = ?
	`
	args := []any{params.ConversationID}
	if beforeID != "" {
		query += `
			AND (
				m.created_at < ?
				OR (m.created_at = ? AND m.id < ?)
			)
		`
		args = append(args, beforeCreatedAtText, beforeCreatedAtText, beforeID)
	}
	query += `
		ORDER BY m.created_at DESC, m.id DESC
		LIMIT ?
	`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list conversation messages: %w", err)
	}
	defer rows.Close()

	messages := make([]models.MessageDetails, 0, limit)
	for rows.Next() {
		var (
			details             models.MessageDetails
			content             sql.NullString
			attachmentURL       sql.NullString
			attachmentType      sql.NullString
			edited              int
			deleted             int
			createdAtText       string
			updatedAtText       string
			senderCreatedAtText string
		)
		if err := rows.Scan(
			&details.Message.ID,
			&details.Message.ConversationID,
			&details.Message.SenderID,
			&content,
			&attachmentURL,
			&attachmentType,
			&edited,
			&deleted,
			&createdAtText,
			&updatedAtText,
			&details.Sender.ID,
			&details.Sender.Username,
			&senderCreatedAtText,
		); err != nil {
			return nil, fmt.Errorf("scan conversation message: %w", err)
		}

		details.Message.Content = nullStringPointer(content)
		details.Message.AttachmentURL = nullStringPointer(attachmentURL)
		if attachmentType.Valid {
			typed := models.AttachmentType(attachmentType.String)
			details.Message.AttachmentType = &typed
		}
		details.Message.Edited = edited != 0
		details.Message.Deleted = deleted != 0

		createdAt, err := parseStoredTime(createdAtText)
		if err != nil {
			return nil, fmt.Errorf("parse message created_at: %w", err)
		}
		details.Message.CreatedAt = createdAt

		updatedAt, err := parseStoredTime(updatedAtText)
		if err != nil {
			return nil, fmt.Errorf("parse message updated_at: %w", err)
		}
		details.Message.UpdatedAt = updatedAt

		senderCreatedAt, err := parseStoredTime(senderCreatedAtText)
		if err != nil {
			return nil, fmt.Errorf("parse message sender created_at: %w", err)
		}
		details.Sender.CreatedAt = senderCreatedAt

		messages = append(messages, details)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation messages: %w", err)
	}

	return messages, nil
}

func (s *SQLiteStore) CreateMessage(ctx context.Context, params models.CreateMessageParams) (models.Message, error) {
	if err := s.ensureConversationParticipant(ctx, params.ConversationID, params.SenderID); err != nil {
		return models.Message{}, err
	}

	const query = `
		INSERT INTO messages (
			id, conversation_id, sender_id, content, attachment_url, attachment_type, edited, deleted, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, 0, 0, ?, ?)
	`

	var attachmentType any
	if params.AttachmentType != nil {
		attachmentType = string(*params.AttachmentType)
	}
	_, err := s.db.ExecContext(
		ctx,
		query,
		params.ID,
		params.ConversationID,
		params.SenderID,
		params.Content,
		params.AttachmentURL,
		attachmentType,
		formatTime(params.CreatedAt),
		formatTime(params.UpdatedAt),
	)
	if err != nil {
		return models.Message{}, fmt.Errorf("insert message: %w", err)
	}

	return s.getMessageByID(ctx, params.ID)
}

func (s *SQLiteStore) UpdateMessage(ctx context.Context, params models.UpdateMessageParams) (models.Message, error) {
	const query = `
		UPDATE messages
		SET content = ?, edited = 1, updated_at = ?
		WHERE id = ? AND sender_id = ?
	`
	res, err := s.db.ExecContext(
		ctx,
		query,
		strings.TrimSpace(params.Content),
		formatTime(params.UpdatedAt),
		params.MessageID,
		params.ActorUserID,
	)
	if err != nil {
		return models.Message{}, fmt.Errorf("update message: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return models.Message{}, fmt.Errorf("update message rows affected: %w", err)
	}
	if rowsAffected == 0 {
		if _, err := s.getMessageByID(ctx, params.MessageID); err != nil {
			return models.Message{}, err
		}
		return models.Message{}, ErrForbidden
	}

	return s.getMessageByID(ctx, params.MessageID)
}

func (s *SQLiteStore) SoftDeleteMessage(ctx context.Context, params models.SoftDeleteMessageParams) (models.Message, error) {
	const query = `
		UPDATE messages
		SET content = NULL, attachment_url = NULL, attachment_type = NULL, deleted = 1, updated_at = ?
		WHERE id = ? AND sender_id = ?
	`
	res, err := s.db.ExecContext(
		ctx,
		query,
		formatTime(params.UpdatedAt),
		params.MessageID,
		params.ActorUserID,
	)
	if err != nil {
		return models.Message{}, fmt.Errorf("soft-delete message: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return models.Message{}, fmt.Errorf("soft-delete message rows affected: %w", err)
	}
	if rowsAffected == 0 {
		if _, err := s.getMessageByID(ctx, params.MessageID); err != nil {
			return models.Message{}, err
		}
		return models.Message{}, ErrForbidden
	}

	return s.getMessageByID(ctx, params.MessageID)
}

func (s *SQLiteStore) ToggleMessageReaction(ctx context.Context, params models.ToggleMessageReactionParams) (models.ToggleReactionResult, error) {
	message, err := s.getMessageByID(ctx, params.MessageID)
	if err != nil {
		return models.ToggleReactionResult{}, err
	}
	if err := s.ensureConversationParticipant(ctx, message.ConversationID, params.ActorUserID); err != nil {
		return models.ToggleReactionResult{}, err
	}

	reactionID := strings.TrimSpace(params.ReactionID)
	emoji := strings.TrimSpace(params.Emoji)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.ToggleReactionResult{}, fmt.Errorf("begin toggle reaction tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	existing, err := getReactionByMessageUserEmojiTx(ctx, tx, params.MessageID, params.ActorUserID, emoji)
	if err == nil {
		if err := deleteReactionByIDTx(ctx, tx, existing.ID); err != nil {
			return models.ToggleReactionResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return models.ToggleReactionResult{}, fmt.Errorf("commit toggle remove reaction tx: %w", err)
		}
		return models.ToggleReactionResult{
			Action:   models.ReactionMutationRemoved,
			Reaction: existing,
		}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return models.ToggleReactionResult{}, err
	}

	const insertQuery = `
		INSERT INTO reactions (id, message_id, user_id, emoji, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = tx.ExecContext(
		ctx,
		insertQuery,
		reactionID,
		params.MessageID,
		params.ActorUserID,
		emoji,
		formatTime(params.CreatedAt),
	)
	if err != nil {
		return models.ToggleReactionResult{}, fmt.Errorf("insert reaction: %w", err)
	}

	addedReaction, err := getReactionByIDTx(ctx, tx, reactionID)
	if err != nil {
		return models.ToggleReactionResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.ToggleReactionResult{}, fmt.Errorf("commit toggle add reaction tx: %w", err)
	}

	return models.ToggleReactionResult{
		Action:   models.ReactionMutationAdded,
		Reaction: addedReaction,
	}, nil
}

func (s *SQLiteStore) RemoveMessageReaction(ctx context.Context, params models.RemoveMessageReactionParams) (models.Reaction, error) {
	message, err := s.getMessageByID(ctx, params.MessageID)
	if err != nil {
		return models.Reaction{}, err
	}
	if err := s.ensureConversationParticipant(ctx, message.ConversationID, params.ActorUserID); err != nil {
		return models.Reaction{}, err
	}

	emoji := strings.TrimSpace(params.Emoji)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Reaction{}, fmt.Errorf("begin remove reaction tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	reaction, err := getReactionByMessageUserEmojiTx(ctx, tx, params.MessageID, params.ActorUserID, emoji)
	if err != nil {
		return models.Reaction{}, err
	}
	if err := deleteReactionByIDTx(ctx, tx, reaction.ID); err != nil {
		return models.Reaction{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.Reaction{}, fmt.Errorf("commit remove reaction tx: %w", err)
	}

	return reaction, nil
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

func (s *SQLiteStore) getMessageByID(ctx context.Context, messageID string) (models.Message, error) {
	const query = `
		SELECT id, conversation_id, sender_id, content, attachment_url, attachment_type, edited, deleted, created_at, updated_at
		FROM messages
		WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, messageID)

	var (
		message        models.Message
		content        sql.NullString
		attachmentURL  sql.NullString
		attachmentType sql.NullString
		edited         int
		deleted        int
		createdAtText  string
		updatedAtText  string
	)
	if err := row.Scan(
		&message.ID,
		&message.ConversationID,
		&message.SenderID,
		&content,
		&attachmentURL,
		&attachmentType,
		&edited,
		&deleted,
		&createdAtText,
		&updatedAtText,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Message{}, ErrNotFound
		}
		return models.Message{}, fmt.Errorf("scan message: %w", err)
	}

	message.Content = nullStringPointer(content)
	message.AttachmentURL = nullStringPointer(attachmentURL)
	if attachmentType.Valid {
		typed := models.AttachmentType(attachmentType.String)
		message.AttachmentType = &typed
	}
	message.Edited = edited != 0
	message.Deleted = deleted != 0

	createdAt, err := parseStoredTime(createdAtText)
	if err != nil {
		return models.Message{}, fmt.Errorf("parse message created_at: %w", err)
	}
	message.CreatedAt = createdAt

	updatedAt, err := parseStoredTime(updatedAtText)
	if err != nil {
		return models.Message{}, fmt.Errorf("parse message updated_at: %w", err)
	}
	message.UpdatedAt = updatedAt

	return message, nil
}

func (s *SQLiteStore) ensureConversationParticipant(ctx context.Context, conversationID, userID string) error {
	conversation, err := s.getConversationByID(ctx, conversationID)
	if err != nil {
		return err
	}
	if conversation.ParticipantA != userID && conversation.ParticipantB != userID {
		return ErrForbidden
	}
	return nil
}

func (s *SQLiteStore) getConversationByID(ctx context.Context, conversationID string) (models.Conversation, error) {
	return getConversationByIDQuery(ctx, s.db.QueryRowContext(ctx, `
		SELECT id, participant_a, participant_b, created_at
		FROM conversations
		WHERE id = ?
	`, conversationID))
}

func getConversationByIDTx(ctx context.Context, tx *sql.Tx, conversationID string) (models.Conversation, error) {
	return getConversationByIDQuery(ctx, tx.QueryRowContext(ctx, `
		SELECT id, participant_a, participant_b, created_at
		FROM conversations
		WHERE id = ?
	`, conversationID))
}

func getReactionByIDTx(ctx context.Context, tx *sql.Tx, reactionID string) (models.Reaction, error) {
	return getReactionByQueryRow(tx.QueryRowContext(ctx, `
		SELECT id, message_id, user_id, emoji, created_at
		FROM reactions
		WHERE id = ?
	`, reactionID))
}

func getReactionByMessageUserEmojiTx(ctx context.Context, tx *sql.Tx, messageID, userID, emoji string) (models.Reaction, error) {
	return getReactionByQueryRow(tx.QueryRowContext(ctx, `
		SELECT id, message_id, user_id, emoji, created_at
		FROM reactions
		WHERE message_id = ? AND user_id = ? AND emoji = ?
	`, messageID, userID, emoji))
}

func deleteReactionByIDTx(ctx context.Context, tx *sql.Tx, reactionID string) error {
	res, err := tx.ExecContext(ctx, `DELETE FROM reactions WHERE id = ?`, reactionID)
	if err != nil {
		return fmt.Errorf("delete reaction: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete reaction rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func getConversationByIDQuery(_ context.Context, row *sql.Row) (models.Conversation, error) {
	var (
		conversation  models.Conversation
		createdAtText string
	)
	if err := row.Scan(
		&conversation.ID,
		&conversation.ParticipantA,
		&conversation.ParticipantB,
		&createdAtText,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Conversation{}, ErrNotFound
		}
		return models.Conversation{}, fmt.Errorf("scan conversation: %w", err)
	}

	createdAt, err := parseStoredTime(createdAtText)
	if err != nil {
		return models.Conversation{}, fmt.Errorf("parse conversation created_at: %w", err)
	}
	conversation.CreatedAt = createdAt
	return conversation, nil
}

func getConversationByParticipantsTx(ctx context.Context, tx *sql.Tx, participantA, participantB string) (models.Conversation, error) {
	return getConversationByIDQuery(ctx, tx.QueryRowContext(ctx, `
		SELECT id, participant_a, participant_b, created_at
		FROM conversations
		WHERE participant_a = ? AND participant_b = ?
	`, participantA, participantB))
}

func nullStringPointer(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func getReactionByQueryRow(row *sql.Row) (models.Reaction, error) {
	var (
		reaction      models.Reaction
		createdAtText string
	)
	if err := row.Scan(
		&reaction.ID,
		&reaction.MessageID,
		&reaction.UserID,
		&reaction.Emoji,
		&createdAtText,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Reaction{}, ErrNotFound
		}
		return models.Reaction{}, fmt.Errorf("scan reaction: %w", err)
	}

	createdAt, err := parseStoredTime(createdAtText)
	if err != nil {
		return models.Reaction{}, fmt.Errorf("parse reaction created_at: %w", err)
	}
	reaction.CreatedAt = createdAt

	return reaction, nil
}

func nullableMessageToModel(
	messageID sql.NullString,
	messageConversationID sql.NullString,
	messageSenderID sql.NullString,
	messageContent sql.NullString,
	messageAttachmentURL sql.NullString,
	messageAttachmentType sql.NullString,
	messageEdited sql.NullInt64,
	messageDeleted sql.NullInt64,
	messageCreatedAtText sql.NullString,
	messageUpdatedAtText sql.NullString,
) (models.Message, error) {
	message := models.Message{
		ID:             messageID.String,
		ConversationID: messageConversationID.String,
		SenderID:       messageSenderID.String,
		Content:        nullStringPointer(messageContent),
		AttachmentURL:  nullStringPointer(messageAttachmentURL),
		Edited:         messageEdited.Int64 != 0,
		Deleted:        messageDeleted.Int64 != 0,
	}
	if messageAttachmentType.Valid {
		typed := models.AttachmentType(messageAttachmentType.String)
		message.AttachmentType = &typed
	}

	if !messageCreatedAtText.Valid || !messageUpdatedAtText.Valid {
		return models.Message{}, errors.New("missing message timestamps")
	}

	createdAt, err := parseStoredTime(messageCreatedAtText.String)
	if err != nil {
		return models.Message{}, err
	}
	message.CreatedAt = createdAt

	updatedAt, err := parseStoredTime(messageUpdatedAtText.String)
	if err != nil {
		return models.Message{}, err
	}
	message.UpdatedAt = updatedAt
	return message, nil
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

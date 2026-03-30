package store

import (
	"context"
	"encoding/json"
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

func TestSQLiteStorePhase2CoreOperations(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "phase2.sqlite")

	s, err := NewSQLiteStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	base := time.Now().UTC().Truncate(time.Microsecond)

	alice := mustCreateUser(t, ctx, s, models.CreateUserParams{
		ID:        "user-alice",
		Username:  "alice",
		PINHash:   "hash",
		CreatedAt: base,
	})
	bob := mustCreateUser(t, ctx, s, models.CreateUserParams{
		ID:        "user-bob",
		Username:  "bob",
		PINHash:   "hash",
		CreatedAt: base.Add(time.Second),
	})
	charlie := mustCreateUser(t, ctx, s, models.CreateUserParams{
		ID:        "user-charlie",
		Username:  "charlie",
		PINHash:   "hash",
		CreatedAt: base.Add(2 * time.Second),
	})
	_ = mustCreateUser(t, ctx, s, models.CreateUserParams{
		ID:        "user-alex",
		Username:  "alex",
		PINHash:   "hash",
		CreatedAt: base.Add(3 * time.Second),
	})

	searchResults, err := s.SearchUsersByUsername(ctx, models.SearchUsersParams{
		Query: "al",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("SearchUsersByUsername() error = %v", err)
	}
	if len(searchResults) != 2 || searchResults[0].Username != "alex" || searchResults[1].Username != "alice" {
		t.Fatalf("unexpected search result order/content: %+v", searchResults)
	}

	conversationAB, err := s.GetOrCreateDirectConversation(ctx, models.GetOrCreateDirectConversationParams{
		ConversationID: "conv-ab-1",
		CurrentUserID:  alice.ID,
		TargetUserID:   bob.ID,
		CreatedAt:      base.Add(4 * time.Second),
	})
	if err != nil {
		t.Fatalf("GetOrCreateDirectConversation() create AB error = %v", err)
	}
	if conversationAB.ID != "conv-ab-1" {
		t.Fatalf("expected conv-ab-1, got %q", conversationAB.ID)
	}

	conversationABSecond, err := s.GetOrCreateDirectConversation(ctx, models.GetOrCreateDirectConversationParams{
		ConversationID: "conv-ab-2",
		CurrentUserID:  bob.ID,
		TargetUserID:   alice.ID,
		CreatedAt:      base.Add(5 * time.Second),
	})
	if err != nil {
		t.Fatalf("GetOrCreateDirectConversation() fetch AB error = %v", err)
	}
	if conversationABSecond.ID != conversationAB.ID {
		t.Fatalf("expected existing AB conversation %q, got %q", conversationAB.ID, conversationABSecond.ID)
	}

	conversationAC, err := s.GetOrCreateDirectConversation(ctx, models.GetOrCreateDirectConversationParams{
		ConversationID: "conv-ac-1",
		CurrentUserID:  alice.ID,
		TargetUserID:   charlie.ID,
		CreatedAt:      base.Add(6 * time.Second),
	})
	if err != nil {
		t.Fatalf("GetOrCreateDirectConversation() create AC error = %v", err)
	}

	detailsAB, err := s.GetConversationByIDForUser(ctx, models.GetConversationForUserParams{
		ConversationID: conversationAB.ID,
		UserID:         alice.ID,
	})
	if err != nil {
		t.Fatalf("GetConversationByIDForUser() error = %v", err)
	}
	if detailsAB.Conversation.ID != conversationAB.ID {
		t.Fatalf("unexpected conversation details: %+v", detailsAB)
	}

	_, err = s.GetConversationByIDForUser(ctx, models.GetConversationForUserParams{
		ConversationID: conversationAB.ID,
		UserID:         charlie.ID,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-participant detail fetch, got %v", err)
	}

	msg1, err := s.CreateMessage(ctx, models.CreateMessageParams{
		ID:             "msg-1",
		ConversationID: conversationAB.ID,
		SenderID:       alice.ID,
		Content:        stringPtr("hello"),
		CreatedAt:      base.Add(7 * time.Second),
		UpdatedAt:      base.Add(7 * time.Second),
	})
	if err != nil {
		t.Fatalf("CreateMessage(msg1) error = %v", err)
	}
	if msg1.ID != "msg-1" || msg1.Content == nil || *msg1.Content != "hello" {
		t.Fatalf("unexpected msg1: %+v", msg1)
	}

	msg2, err := s.CreateMessage(ctx, models.CreateMessageParams{
		ID:             "msg-2",
		ConversationID: conversationAB.ID,
		SenderID:       bob.ID,
		Content:        stringPtr("hi alice"),
		CreatedAt:      base.Add(8 * time.Second),
		UpdatedAt:      base.Add(8 * time.Second),
	})
	if err != nil {
		t.Fatalf("CreateMessage(msg2) error = %v", err)
	}

	imageType := models.AttachmentTypeImage
	msg3, err := s.CreateMessage(ctx, models.CreateMessageParams{
		ID:             "msg-3",
		ConversationID: conversationAB.ID,
		SenderID:       alice.ID,
		AttachmentURL:  stringPtr("/static/uploads/sample.png"),
		AttachmentType: &imageType,
		CreatedAt:      base.Add(9 * time.Second),
		UpdatedAt:      base.Add(9 * time.Second),
	})
	if err != nil {
		t.Fatalf("CreateMessage(msg3) error = %v", err)
	}
	if msg3.AttachmentURL == nil || msg3.AttachmentType == nil || *msg3.AttachmentType != models.AttachmentTypeImage {
		t.Fatalf("expected attachment metadata on msg3, got %+v", msg3)
	}

	jsonRenderSpec := json.RawMessage(`{"root":"stack-1","elements":{"stack-1":{"type":"Stack"}}}`)
	msg4, err := s.CreateMessage(ctx, models.CreateMessageParams{
		ID:             "msg-4",
		ConversationID: conversationAB.ID,
		SenderID:       alice.ID,
		Kind:           models.MessageKindJSONRender,
		JSONRenderSpec: jsonRenderSpec,
		CreatedAt:      base.Add(9*time.Second + time.Millisecond),
		UpdatedAt:      base.Add(9*time.Second + time.Millisecond),
	})
	if err != nil {
		t.Fatalf("CreateMessage(msg4) error = %v", err)
	}
	if msg4.Kind != models.MessageKindJSONRender || string(msg4.JSONRenderSpec) != string(jsonRenderSpec) {
		t.Fatalf("expected json_render metadata on msg4, got %+v", msg4)
	}

	_, err = s.CreateMessage(ctx, models.CreateMessageParams{
		ID:             "msg-forbidden",
		ConversationID: conversationAB.ID,
		SenderID:       charlie.ID,
		Content:        stringPtr("not allowed"),
		CreatedAt:      base.Add(10 * time.Second),
		UpdatedAt:      base.Add(10 * time.Second),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden when non-participant sends message, got %v", err)
	}

	conversations, err := s.ListConversationsByUser(ctx, models.ListUserConversationsParams{
		UserID: alice.ID,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("ListConversationsByUser() error = %v", err)
	}
	if len(conversations) != 2 {
		t.Fatalf("expected 2 conversations for alice, got %d", len(conversations))
	}
	if conversations[0].Conversation.ID != conversationAB.ID {
		t.Fatalf("expected AB first by recency, got %q", conversations[0].Conversation.ID)
	}
	if conversations[0].OtherUser.ID != bob.ID {
		t.Fatalf("expected other user bob, got %+v", conversations[0].OtherUser)
	}
	if conversations[0].LastMessage == nil || conversations[0].LastMessage.ID != msg4.ID {
		t.Fatalf("expected last message msg4 in AB summary, got %+v", conversations[0].LastMessage)
	}
	if conversations[1].Conversation.ID != conversationAC.ID {
		t.Fatalf("expected AC second, got %q", conversations[1].Conversation.ID)
	}
	if conversations[1].LastMessage != nil {
		t.Fatalf("expected no last message for AC, got %+v", conversations[1].LastMessage)
	}

	firstPage, err := s.ListMessagesByConversation(ctx, models.ListConversationMessagesParams{
		ConversationID: conversationAB.ID,
		UserID:         alice.ID,
		Limit:          2,
	})
	if err != nil {
		t.Fatalf("ListMessagesByConversation(first page) error = %v", err)
	}
	if len(firstPage) != 2 || firstPage[0].Message.ID != "msg-4" || firstPage[1].Message.ID != "msg-3" {
		t.Fatalf("unexpected first message page: %+v", firstPage)
	}

	before := "msg-2"
	secondPage, err := s.ListMessagesByConversation(ctx, models.ListConversationMessagesParams{
		ConversationID:  conversationAB.ID,
		UserID:          bob.ID,
		BeforeMessageID: &before,
		Limit:           20,
	})
	if err != nil {
		t.Fatalf("ListMessagesByConversation(second page) error = %v", err)
	}
	if len(secondPage) != 1 || secondPage[0].Message.ID != "msg-1" {
		t.Fatalf("unexpected second message page: %+v", secondPage)
	}

	_, err = s.ListMessagesByConversation(ctx, models.ListConversationMessagesParams{
		ConversationID: conversationAB.ID,
		UserID:         charlie.ID,
		Limit:          20,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden when outsider lists messages, got %v", err)
	}

	edited, err := s.UpdateMessage(ctx, models.UpdateMessageParams{
		MessageID:   "msg-1",
		ActorUserID: alice.ID,
		Content:     "hello edited",
		UpdatedAt:   base.Add(11 * time.Second),
	})
	if err != nil {
		t.Fatalf("UpdateMessage() error = %v", err)
	}
	if edited.Content == nil || *edited.Content != "hello edited" || !edited.Edited {
		t.Fatalf("unexpected edited message: %+v", edited)
	}

	_, err = s.UpdateMessage(ctx, models.UpdateMessageParams{
		MessageID:   "msg-1",
		ActorUserID: bob.ID,
		Content:     "not owner",
		UpdatedAt:   base.Add(12 * time.Second),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-owner edit, got %v", err)
	}

	deleted, err := s.SoftDeleteMessage(ctx, models.SoftDeleteMessageParams{
		MessageID:   msg2.ID,
		ActorUserID: bob.ID,
		UpdatedAt:   base.Add(13 * time.Second),
	})
	if err != nil {
		t.Fatalf("SoftDeleteMessage() error = %v", err)
	}
	if !deleted.Deleted || deleted.Content != nil || deleted.AttachmentURL != nil || deleted.AttachmentType != nil {
		t.Fatalf("unexpected deleted message payload: %+v", deleted)
	}

	_, err = s.SoftDeleteMessage(ctx, models.SoftDeleteMessageParams{
		MessageID:   msg2.ID,
		ActorUserID: alice.ID,
		UpdatedAt:   base.Add(14 * time.Second),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-owner soft delete, got %v", err)
	}
}

func TestSQLiteStoreReactionPersistence(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "reactions.sqlite")

	s, err := NewSQLiteStore(ctx, dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = s.Close()
	})

	base := time.Now().UTC().Truncate(time.Microsecond)

	alice := mustCreateUser(t, ctx, s, models.CreateUserParams{
		ID:        "user-alice",
		Username:  "alice",
		PINHash:   "hash",
		CreatedAt: base,
	})
	bob := mustCreateUser(t, ctx, s, models.CreateUserParams{
		ID:        "user-bob",
		Username:  "bob",
		PINHash:   "hash",
		CreatedAt: base.Add(time.Second),
	})
	charlie := mustCreateUser(t, ctx, s, models.CreateUserParams{
		ID:        "user-charlie",
		Username:  "charlie",
		PINHash:   "hash",
		CreatedAt: base.Add(2 * time.Second),
	})

	conversationAB, err := s.GetOrCreateDirectConversation(ctx, models.GetOrCreateDirectConversationParams{
		ConversationID: "conv-ab",
		CurrentUserID:  alice.ID,
		TargetUserID:   bob.ID,
		CreatedAt:      base.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("GetOrCreateDirectConversation() error = %v", err)
	}

	message, err := s.CreateMessage(ctx, models.CreateMessageParams{
		ID:             "msg-1",
		ConversationID: conversationAB.ID,
		SenderID:       alice.ID,
		Content:        stringPtr("hello"),
		CreatedAt:      base.Add(4 * time.Second),
		UpdatedAt:      base.Add(4 * time.Second),
	})
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}

	firstToggle, err := s.ToggleMessageReaction(ctx, models.ToggleMessageReactionParams{
		ReactionID:  "rxn-1",
		MessageID:   message.ID,
		ActorUserID: alice.ID,
		Emoji:       "👍",
		CreatedAt:   base.Add(5 * time.Second),
	})
	if err != nil {
		t.Fatalf("ToggleMessageReaction(add) error = %v", err)
	}
	if firstToggle.Action != models.ReactionMutationAdded {
		t.Fatalf("expected add action, got %q", firstToggle.Action)
	}
	if firstToggle.Reaction.ID != "rxn-1" || firstToggle.Reaction.MessageID != message.ID || firstToggle.Reaction.UserID != alice.ID || firstToggle.Reaction.Emoji != "👍" {
		t.Fatalf("unexpected added reaction: %+v", firstToggle.Reaction)
	}

	secondToggle, err := s.ToggleMessageReaction(ctx, models.ToggleMessageReactionParams{
		ReactionID:  "rxn-2",
		MessageID:   message.ID,
		ActorUserID: alice.ID,
		Emoji:       "👍",
		CreatedAt:   base.Add(6 * time.Second),
	})
	if err != nil {
		t.Fatalf("ToggleMessageReaction(remove) error = %v", err)
	}
	if secondToggle.Action != models.ReactionMutationRemoved {
		t.Fatalf("expected remove action, got %q", secondToggle.Action)
	}
	if secondToggle.Reaction.ID != "rxn-1" || secondToggle.Reaction.UserID != alice.ID || secondToggle.Reaction.Emoji != "👍" {
		t.Fatalf("unexpected removed reaction payload: %+v", secondToggle.Reaction)
	}

	if _, err := s.ToggleMessageReaction(ctx, models.ToggleMessageReactionParams{
		ReactionID:  "rxn-3",
		MessageID:   message.ID,
		ActorUserID: alice.ID,
		Emoji:       "😄",
		CreatedAt:   base.Add(7 * time.Second),
	}); err != nil {
		t.Fatalf("ToggleMessageReaction(add second emoji) error = %v", err)
	}

	bobToggle, err := s.ToggleMessageReaction(ctx, models.ToggleMessageReactionParams{
		ReactionID:  "rxn-4",
		MessageID:   message.ID,
		ActorUserID: bob.ID,
		Emoji:       "👍",
		CreatedAt:   base.Add(8 * time.Second),
	})
	if err != nil {
		t.Fatalf("ToggleMessageReaction(add by bob) error = %v", err)
	}
	if bobToggle.Action != models.ReactionMutationAdded || bobToggle.Reaction.ID != "rxn-4" {
		t.Fatalf("unexpected bob toggle result: %+v", bobToggle)
	}

	var reactionCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reactions WHERE message_id = ?`, message.ID).Scan(&reactionCount); err != nil {
		t.Fatalf("count reactions: %v", err)
	}
	if reactionCount != 2 {
		t.Fatalf("expected 2 reactions after toggles, got %d", reactionCount)
	}

	removed, err := s.RemoveMessageReaction(ctx, models.RemoveMessageReactionParams{
		MessageID:   message.ID,
		ActorUserID: bob.ID,
		Emoji:       "👍",
	})
	if err != nil {
		t.Fatalf("RemoveMessageReaction() error = %v", err)
	}
	if removed.ID != "rxn-4" || removed.UserID != bob.ID || removed.Emoji != "👍" {
		t.Fatalf("unexpected removed reaction: %+v", removed)
	}

	_, err = s.RemoveMessageReaction(ctx, models.RemoveMessageReactionParams{
		MessageID:   message.ID,
		ActorUserID: bob.ID,
		Emoji:       "👍",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for removing non-existent own reaction, got %v", err)
	}

	_, err = s.ToggleMessageReaction(ctx, models.ToggleMessageReactionParams{
		ReactionID:  "rxn-5",
		MessageID:   message.ID,
		ActorUserID: charlie.ID,
		Emoji:       "🔥",
		CreatedAt:   base.Add(9 * time.Second),
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for outsider toggle, got %v", err)
	}

	_, err = s.RemoveMessageReaction(ctx, models.RemoveMessageReactionParams{
		MessageID:   message.ID,
		ActorUserID: charlie.ID,
		Emoji:       "😄",
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for outsider remove, got %v", err)
	}

	_, err = s.ToggleMessageReaction(ctx, models.ToggleMessageReactionParams{
		ReactionID:  "rxn-6",
		MessageID:   "missing-message",
		ActorUserID: alice.ID,
		Emoji:       "🔥",
		CreatedAt:   base.Add(10 * time.Second),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing message toggle, got %v", err)
	}

	_, err = s.RemoveMessageReaction(ctx, models.RemoveMessageReactionParams{
		MessageID:   "missing-message",
		ActorUserID: alice.ID,
		Emoji:       "🔥",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing message remove, got %v", err)
	}
}

func mustCreateUser(t *testing.T, ctx context.Context, s *SQLiteStore, params models.CreateUserParams) models.User {
	t.Helper()
	user, err := s.CreateUser(ctx, params)
	if err != nil {
		t.Fatalf("CreateUser(%s) error = %v", params.ID, err)
	}
	return user
}

func stringPtr(v string) *string {
	return &v
}

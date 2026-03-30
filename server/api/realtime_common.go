package api

import (
	"net/http"
	"strings"

	"agent-messenger/server/models"
	"agent-messenger/server/store"
)

const (
	realtimeSendBufferSize             = 16
	realtimeConversationBootstrapLimit = 1000
)

func listConversationIDsForUser(r *http.Request, dataStore store.Store, userID string) ([]string, error) {
	summaries, err := dataStore.ListConversationsByUser(r.Context(), models.ListUserConversationsParams{
		UserID: userID,
		Limit:  realtimeConversationBootstrapLimit,
	})
	if err != nil {
		return nil, err
	}

	conversationIDs := make([]string, 0, len(summaries))
	seen := make(map[string]struct{}, len(summaries))
	for _, summary := range summaries {
		conversationID := strings.TrimSpace(summary.Conversation.ID)
		if conversationID == "" {
			continue
		}
		if _, ok := seen[conversationID]; ok {
			continue
		}
		seen[conversationID] = struct{}{}
		conversationIDs = append(conversationIDs, conversationID)
	}
	return conversationIDs, nil
}

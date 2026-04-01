package push

type MessageNotification struct {
	ConversationID string
	MessageID      string
	SenderID       string
	SenderName     string
	Preview        string
	URL            string
}

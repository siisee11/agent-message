package models

import (
	"encoding/json"
	"time"
)

type MessageKind string

type AttachmentType string

type MessageAttachment struct {
	URL  string         `json:"url"`
	Type AttachmentType `json:"type"`
}

const (
	MessageKindText       MessageKind = "text"
	MessageKindJSONRender MessageKind = "json_render"

	AttachmentTypeImage AttachmentType = "image"
	AttachmentTypeFile  AttachmentType = "file"
)

// Message is a persisted chat message.
type Message struct {
	ID             string              `json:"id" db:"id"`
	ConversationID string              `json:"conversation_id" db:"conversation_id"`
	SenderID       string              `json:"sender_id" db:"sender_id"`
	Content        *string             `json:"content,omitempty" db:"content"`
	Kind           MessageKind         `json:"kind,omitempty" db:"kind"`
	JSONRenderSpec json.RawMessage     `json:"json_render_spec,omitempty" db:"json_render_spec"`
	Attachments    []MessageAttachment `json:"attachments,omitempty" db:"-"`
	AttachmentURL  *string             `json:"attachment_url,omitempty" db:"attachment_url"`
	AttachmentType *AttachmentType     `json:"attachment_type,omitempty" db:"attachment_type"`
	Edited         bool                `json:"edited" db:"edited"`
	Deleted        bool                `json:"deleted" db:"deleted"`
	CreatedAt      time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at" db:"updated_at"`
}

func (m *Message) ApplyAttachmentFallbacks() {
	if len(m.Attachments) == 0 {
		if m.AttachmentURL != nil && m.AttachmentType != nil {
			m.Attachments = []MessageAttachment{{
				URL:  *m.AttachmentURL,
				Type: *m.AttachmentType,
			}}
		}
		return
	}

	firstAttachment := m.Attachments[0]
	m.AttachmentURL = &firstAttachment.URL
	m.AttachmentType = &firstAttachment.Type
}

package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"agent-message/server/models"
)

func encodeMessageAttachments(attachments []models.MessageAttachment) (any, error) {
	if len(attachments) == 0 {
		return nil, nil
	}

	encoded, err := json.Marshal(attachments)
	if err != nil {
		return nil, fmt.Errorf("marshal attachments: %w", err)
	}

	return string(encoded), nil
}

func applyMessageAttachments(
	message *models.Message,
	attachmentsJSON sql.NullString,
	attachmentURL sql.NullString,
	attachmentType sql.NullString,
) error {
	if attachmentsJSON.Valid {
		var attachments []models.MessageAttachment
		if err := json.Unmarshal([]byte(attachmentsJSON.String), &attachments); err != nil {
			return fmt.Errorf("unmarshal attachments: %w", err)
		}
		message.Attachments = attachments
	}

	message.AttachmentURL = nullStringPointer(attachmentURL)
	if attachmentType.Valid {
		typed := models.AttachmentType(attachmentType.String)
		message.AttachmentType = &typed
	}
	message.ApplyAttachmentFallbacks()
	return nil
}

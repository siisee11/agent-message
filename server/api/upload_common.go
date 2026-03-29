package api

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"agent-messenger/server/models"

	"github.com/google/uuid"
)

const (
	maxUploadBytes          = 20 << 20
	multipartBodySizeBuffer = 1 << 20
	defaultUploadDir        = "./uploads"
	staticUploadsPrefix     = "/static/uploads/"
)

var errRequestEntityTooLarge = errors.New("request entity too large")

func saveUploadedFile(uploadDir string, file io.Reader, header *multipart.FileHeader) (string, models.AttachmentType, error) {
	if header.Size > maxUploadBytes {
		return "", "", errRequestEntityTooLarge
	}

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", "", fmt.Errorf("prepare upload dir: %w", err)
	}

	extension := filepath.Ext(strings.TrimSpace(header.Filename))
	storedName := uuid.NewString() + extension
	destPath := filepath.Join(uploadDir, storedName)

	dest, err := os.Create(destPath)
	if err != nil {
		return "", "", fmt.Errorf("create uploaded file: %w", err)
	}
	defer dest.Close()

	reader := io.LimitReader(file, maxUploadBytes+1)
	written, err := io.Copy(dest, reader)
	if err != nil {
		return "", "", fmt.Errorf("store uploaded file: %w", err)
	}
	if written > maxUploadBytes {
		_ = os.Remove(destPath)
		return "", "", errRequestEntityTooLarge
	}

	attachmentType := models.AttachmentTypeFile
	contentType := strings.ToLower(strings.TrimSpace(header.Header.Get("Content-Type")))
	if strings.HasPrefix(contentType, "image/") {
		attachmentType = models.AttachmentTypeImage
	}

	return staticUploadsPrefix + storedName, attachmentType, nil
}

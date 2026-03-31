package api

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"agent-message/server/models"

	"github.com/google/uuid"
)

const (
	maxUploadBytes          = 20 << 20
	multipartBodySizeBuffer = 1 << 20
	defaultUploadDir        = "./uploads"
	staticUploadsPrefix     = "/static/uploads/"
)

var errRequestEntityTooLarge = errors.New("request entity too large")
var errUnsupportedUploadType = errors.New("unsupported file type")

var allowedImageExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
var allowedFileExtensions = []string{
	".pdf", ".txt", ".zip", ".csv", ".json", ".md",
	".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
	".rtf", ".gz", ".tar",
}
var allowedUploadContentTypes = map[string]struct{}{
	"application/octet-stream":     {},
	"application/pdf":              {},
	"application/zip":              {},
	"application/x-zip-compressed": {},
	"application/gzip":             {},
	"application/x-gzip":           {},
	"application/json":             {},
	"text/plain":                   {},
	"text/csv":                     {},
	"text/markdown":                {},
	"text/rtf":                     {},
	"application/rtf":              {},
	"application/msword":           {},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {},
	"application/vnd.ms-excel": {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":                  {},
	"application/vnd.ms-powerpoint":                                                      {},
	"application/vnd.openxmlformats-officedocument.presentationml.presentation":          {},
	"application/vnd.openxmlformats-officedocument.presentationml.slideshow":             {},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.template":            {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.template":               {},
	"application/vnd.openxmlformats-officedocument.presentationml.template":              {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml":         {},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml":   {},
	"application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml": {},
	"application/vnd.openxmlformats-officedocument.presentationml.slideshow.main+xml":    {},
	"application/vnd.openxmlformats-officedocument.presentationml.template.main+xml":     {},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.template.main+xml":   {},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.template.main+xml":      {},
}

func saveUploadedFile(uploadDir string, file io.Reader, header *multipart.FileHeader) (string, models.AttachmentType, error) {
	if header.Size > maxUploadBytes {
		return "", "", errRequestEntityTooLarge
	}
	if err := validateUploadHeader(header); err != nil {
		return "", "", err
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
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		contentType = strings.ToLower(strings.TrimSpace(mediaType))
	}
	if strings.HasPrefix(contentType, "image/") {
		attachmentType = models.AttachmentTypeImage
	}

	return staticUploadsPrefix + storedName, attachmentType, nil
}

func validateUploadHeader(header *multipart.FileHeader) error {
	filename := strings.TrimSpace(filepath.Base(header.Filename))
	if filename == "" || filename == "." || filename == string(filepath.Separator) {
		return errUnsupportedUploadType
	}

	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	if ext == "" {
		return errUnsupportedUploadType
	}

	isImageExt := slices.Contains(allowedImageExtensions, ext)
	isFileExt := slices.Contains(allowedFileExtensions, ext)
	if !isImageExt && !isFileExt {
		return errUnsupportedUploadType
	}

	rawContentType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if rawContentType == "" {
		return nil
	}

	mediaType, _, err := mime.ParseMediaType(rawContentType)
	if err != nil {
		return errUnsupportedUploadType
	}
	contentType := strings.ToLower(strings.TrimSpace(mediaType))

	if strings.HasPrefix(contentType, "image/") {
		if !isImageExt {
			return errUnsupportedUploadType
		}
		return nil
	}

	if contentType == "application/octet-stream" {
		return nil
	}

	if _, ok := allowedUploadContentTypes[contentType]; ok && isFileExt {
		return nil
	}

	return errUnsupportedUploadType
}

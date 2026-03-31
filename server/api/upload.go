package api

import (
	"errors"
	"net/http"
	"strings"

	"agent-message/server/models"
)

type uploadHandler struct {
	uploadDir string
}

func newUploadHandler(uploadDir string) *uploadHandler {
	trimmed := strings.TrimSpace(uploadDir)
	if trimmed == "" {
		trimmed = defaultUploadDir
	}
	return &uploadHandler{
		uploadDir: trimmed,
	}
}

func (h *uploadHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	if _, ok := userFromContext(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	maxBodyBytes := int64(maxUploadBytes + multipartBodySizeBuffer)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := r.ParseMultipartForm(maxBodyBytes); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "file exceeds 20 MB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if errors.Is(err, http.ErrMissingFile) {
		file, header, err = r.FormFile("attachment")
	}
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			writeError(w, http.StatusBadRequest, "missing file form field")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid file payload")
		return
	}
	defer file.Close()

	url, _, err := saveUploadedFile(h.uploadDir, file, header)
	if err != nil {
		if errors.Is(err, errRequestEntityTooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "file exceeds 20 MB")
			return
		}
		if errors.Is(err, errUnsupportedUploadType) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to upload file")
		return
	}

	writeJSON(w, http.StatusCreated, models.UploadResponse{
		URL: url,
	})
}

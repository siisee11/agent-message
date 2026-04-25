package cmd

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

func TestRunUploadFilePostsMultipartFile(t *testing.T) {
	t.Parallel()

	uploadPath := createTestAttachmentFile(t, "diagram.png", []byte("png-bytes"))
	rt, stdout, _ := newTestRuntime(t, "http://example.test", "tok-upload", func(req *http.Request, body []byte) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.Path != "/api/upload" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.Path)
		}
		mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("parse content type: %v", err)
		}
		if got, want := mediaType, "multipart/form-data"; got != want {
			t.Fatalf("content type mismatch: got %q want %q", got, want)
		}

		reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		part, err := reader.NextPart()
		if err != nil {
			t.Fatalf("read multipart part: %v", err)
		}
		if got, want := part.FormName(), "file"; got != want {
			t.Fatalf("field mismatch: got %q want %q", got, want)
		}
		if got, want := part.FileName(), "diagram.png"; got != want {
			t.Fatalf("filename mismatch: got %q want %q", got, want)
		}
		partBody, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read multipart part body: %v", err)
		}
		if got, want := string(partBody), "png-bytes"; got != want {
			t.Fatalf("body mismatch: got %q want %q", got, want)
		}
		if _, err := reader.NextPart(); err != io.EOF {
			t.Fatalf("expected multipart EOF, got %v", err)
		}

		return jsonResponse(http.StatusCreated, `{"url":"/static/uploads/diagram.png"}`), nil
	})

	if err := runUploadFile(rt, uploadPath); err != nil {
		t.Fatalf("runUploadFile: %v", err)
	}
	if got, want := strings.TrimSpace(stdout.String()), "/static/uploads/diagram.png"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

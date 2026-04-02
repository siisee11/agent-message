package cmd

import (
	"net/http"
	"strings"
	"testing"
)

func TestRunCatalogPromptPrintsPromptFromAPI(t *testing.T) {
	t.Parallel()

	rt, stdout, _ := newTestRuntime(t, "http://example.test", "", func(req *http.Request, _ []byte) (*http.Response, error) {
		if got, want := req.Method, http.MethodGet; got != want {
			t.Fatalf("method mismatch: got %q want %q", got, want)
		}
		if got, want := req.URL.Path, "/api/catalog/prompt"; got != want {
			t.Fatalf("path mismatch: got %q want %q", got, want)
		}

		return jsonResponse(http.StatusOK, `{"prompt":"line 1\nline 2"}`), nil
	})

	if err := runCatalogPrompt(rt); err != nil {
		t.Fatalf("runCatalogPrompt: %v", err)
	}

	if got, want := strings.TrimSpace(stdout.String()), "line 1\nline 2"; got != want {
		t.Fatalf("stdout mismatch: got %q want %q", got, want)
	}
}

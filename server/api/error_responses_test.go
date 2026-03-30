package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIErrorResponsesUseJSONEnvelope(t *testing.T) {
	router, _ := newTestRouter(t)
	alice := registerAndLoginUser(t, router, "alice", "1234")

	tests := []struct {
		name           string
		method         string
		path           string
		token          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "unknown api route",
			method:         http.MethodGet,
			path:           "/api/does-not-exist",
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
		},
		{
			name:           "invalid conversation detail path",
			method:         http.MethodGet,
			path:           "/api/conversations/invalid/path",
			token:          alice.Token,
			expectedStatus: http.StatusNotFound,
			expectedError:  "conversation not found",
		},
		{
			name:           "invalid message path",
			method:         http.MethodPatch,
			path:           "/api/messages/invalid/path",
			token:          alice.Token,
			expectedStatus: http.StatusNotFound,
			expectedError:  "message not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d body=%s", tt.expectedStatus, resp.Code, resp.Body.String())
			}
			if contentType := resp.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
				t.Fatalf("expected application/json content type, got %q", contentType)
			}
			assertErrorBody(t, resp.Body, tt.expectedError)
		})
	}
}

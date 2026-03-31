package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-message/server/models"
)

func TestUsersEndpoints(t *testing.T) {
	router, _ := newTestRouter(t)

	alice := registerAndLoginUser(t, router, "alice", "1234")
	_ = registerAndLoginUser(t, router, "bob", "1234")
	_ = registerAndLoginUser(t, router, "charlie", "1234")

	t.Run("me requires auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected %d, got %d", http.StatusUnauthorized, resp.Code)
		}
	})

	t.Run("get me returns current profile", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users/me", nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var profile models.UserProfile
		if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
			t.Fatalf("decode me response: %v", err)
		}
		if profile.ID != alice.User.ID || profile.Username != "alice" {
			t.Fatalf("unexpected profile: %+v", profile)
		}
	})

	t.Run("search users by username", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users?username=bo", nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var profiles []models.UserProfile
		if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
			t.Fatalf("decode users response: %v", err)
		}
		if len(profiles) != 1 || profiles[0].Username != "bob" {
			t.Fatalf("unexpected users response: %+v", profiles)
		}
	})

	t.Run("search users validates limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users?username=bo&limit=0", nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.Code)
		}
	})

	t.Run("search users validates username query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users?username=bo%20x", nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, resp.Code)
		}
	})

	t.Run("search users excludes self", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/users?username=ali", nil)
		req.Header.Set("Authorization", "Bearer "+alice.Token)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected %d, got %d body=%s", http.StatusOK, resp.Code, resp.Body.String())
		}

		var profiles []models.UserProfile
		if err := json.NewDecoder(resp.Body).Decode(&profiles); err != nil {
			t.Fatalf("decode users response: %v", err)
		}
		if len(profiles) != 0 {
			t.Fatalf("expected no results after self-filtering, got %+v", profiles)
		}
	})
}

func TestConversationsEndpoints(t *testing.T) {
	router, _ := newTestRouter(t)

	alice := registerAndLoginUser(t, router, "alice", "1234")
	bob := registerAndLoginUser(t, router, "bob", "1234")
	charlie := registerAndLoginUser(t, router, "charlie", "1234")

	startConversation := func(token, username string) (int, models.ConversationDetails) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/conversations", bytes.NewBufferString(`{"username":"`+username+`"}`))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		var details models.ConversationDetails
		if resp.Code == http.StatusOK || resp.Code == http.StatusCreated {
			if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
				t.Fatalf("decode conversation response: %v", err)
			}
		}

		return resp.Code, details
	}

	status, created := startConversation(alice.Token, "bob")
	if status != http.StatusCreated {
		t.Fatalf("expected first start conversation status %d, got %d", http.StatusCreated, status)
	}
	if created.Conversation.ID == "" {
		t.Fatalf("expected conversation id in creation response")
	}

	status, existing := startConversation(alice.Token, "bob")
	if status != http.StatusOK {
		t.Fatalf("expected second start conversation status %d, got %d", http.StatusOK, status)
	}
	if existing.Conversation.ID != created.Conversation.ID {
		t.Fatalf("expected existing conversation %q, got %q", created.Conversation.ID, existing.Conversation.ID)
	}

	reqList := httptest.NewRequest(http.MethodGet, "/api/conversations", nil)
	reqList.Header.Set("Authorization", "Bearer "+alice.Token)
	respList := httptest.NewRecorder()
	router.ServeHTTP(respList, reqList)
	if respList.Code != http.StatusOK {
		t.Fatalf("expected list status %d, got %d body=%s", http.StatusOK, respList.Code, respList.Body.String())
	}

	var summaries []models.ConversationSummary
	if err := json.NewDecoder(respList.Body).Decode(&summaries); err != nil {
		t.Fatalf("decode conversation list: %v", err)
	}
	if len(summaries) != 1 || summaries[0].Conversation.ID != created.Conversation.ID {
		t.Fatalf("unexpected conversation list: %+v", summaries)
	}
	if summaries[0].OtherUser.ID != bob.User.ID {
		t.Fatalf("expected bob as other user, got %+v", summaries[0].OtherUser)
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/api/conversations/"+created.Conversation.ID, nil)
	reqGet.Header.Set("Authorization", "Bearer "+alice.Token)
	respGet := httptest.NewRecorder()
	router.ServeHTTP(respGet, reqGet)
	if respGet.Code != http.StatusOK {
		t.Fatalf("expected get status %d, got %d body=%s", http.StatusOK, respGet.Code, respGet.Body.String())
	}

	var details models.ConversationDetails
	if err := json.NewDecoder(respGet.Body).Decode(&details); err != nil {
		t.Fatalf("decode conversation details: %v", err)
	}
	if details.Conversation.ID != created.Conversation.ID {
		t.Fatalf("unexpected conversation details: %+v", details)
	}

	reqForbidden := httptest.NewRequest(http.MethodGet, "/api/conversations/"+created.Conversation.ID, nil)
	reqForbidden.Header.Set("Authorization", "Bearer "+charlie.Token)
	respForbidden := httptest.NewRecorder()
	router.ServeHTTP(respForbidden, reqForbidden)
	if respForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden status %d, got %d", http.StatusForbidden, respForbidden.Code)
	}

	reqUnknown := httptest.NewRequest(http.MethodPost, "/api/conversations", bytes.NewBufferString(`{"username":"nobody"}`))
	reqUnknown.Header.Set("Authorization", "Bearer "+alice.Token)
	reqUnknown.Header.Set("Content-Type", "application/json")
	respUnknown := httptest.NewRecorder()
	router.ServeHTTP(respUnknown, reqUnknown)
	if respUnknown.Code != http.StatusNotFound {
		t.Fatalf("expected not found status %d, got %d", http.StatusNotFound, respUnknown.Code)
	}

	reqSelf := httptest.NewRequest(http.MethodPost, "/api/conversations", bytes.NewBufferString(`{"username":"alice"}`))
	reqSelf.Header.Set("Authorization", "Bearer "+alice.Token)
	reqSelf.Header.Set("Content-Type", "application/json")
	respSelf := httptest.NewRecorder()
	router.ServeHTTP(respSelf, reqSelf)
	if respSelf.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request status %d, got %d", http.StatusBadRequest, respSelf.Code)
	}

	reqListInvalidLimit := httptest.NewRequest(http.MethodGet, "/api/conversations?limit=0", nil)
	reqListInvalidLimit.Header.Set("Authorization", "Bearer "+alice.Token)
	respListInvalidLimit := httptest.NewRecorder()
	router.ServeHTTP(respListInvalidLimit, reqListInvalidLimit)
	if respListInvalidLimit.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request status %d for invalid limit, got %d", http.StatusBadRequest, respListInvalidLimit.Code)
	}

	reqNotFound := httptest.NewRequest(http.MethodGet, "/api/conversations/does-not-exist", nil)
	reqNotFound.Header.Set("Authorization", "Bearer "+alice.Token)
	respNotFound := httptest.NewRecorder()
	router.ServeHTTP(respNotFound, reqNotFound)
	if respNotFound.Code != http.StatusNotFound {
		t.Fatalf("expected not found status %d, got %d", http.StatusNotFound, respNotFound.Code)
	}
}

func registerAndLoginUser(t *testing.T, router http.Handler, username, pin string) models.AuthResponse {
	t.Helper()

	body := `{"username":"` + username + `","pin":"` + pin + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("register %s expected %d, got %d body=%s", username, http.StatusCreated, resp.Code, resp.Body.String())
	}

	var result models.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode auth response: %v", err)
	}
	return result
}

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const defaultHTTPTimeout = 15 * time.Second

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if strings.TrimSpace(e.Message) != "" {
		return fmt.Sprintf("api request failed (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("api request failed (%d)", e.StatusCode)
}

type errorPayload struct {
	Error string `json:"error"`
}

// Client is a REST client for the Agent Messenger server contract.
type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	token      string
}

func NewClient(serverURL, token string) (*Client, error) {
	c := &Client{
		httpClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
	if err := c.SetServerURL(serverURL); err != nil {
		return nil, err
	}
	c.SetToken(token)
	return c, nil
}

func (c *Client) SetServerURL(serverURL string) error {
	raw := strings.TrimSpace(serverURL)
	if raw == "" {
		return errors.New("server URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse server URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("server URL must start with http:// or https://")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return errors.New("server URL host is required")
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	parsed.RawPath = strings.TrimSuffix(parsed.RawPath, "/")
	c.baseURL = parsed
	return nil
}

func (c *Client) ServerURL() string {
	if c.baseURL == nil {
		return ""
	}
	return c.baseURL.String()
}

func (c *Client) SetToken(token string) {
	c.token = strings.TrimSpace(token)
}

func (c *Client) Token() string {
	return c.token
}

func (c *Client) SetHTTPClient(httpClient *http.Client) {
	if httpClient == nil {
		return
	}
	c.httpClient = httpClient
}

func (c *Client) Register(ctx context.Context, username, pin string) (AuthResponse, error) {
	var out AuthResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/auth/register", map[string]string{
		"username": username,
		"pin":      pin,
	}, &out)
	return out, err
}

func (c *Client) Login(ctx context.Context, username, pin string) (AuthResponse, error) {
	var out AuthResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/auth/login", map[string]string{
		"username": username,
		"pin":      pin,
	}, &out)
	return out, err
}

func (c *Client) Logout(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/auth/logout", nil, nil)
}

func (c *Client) Me(ctx context.Context) (UserProfile, error) {
	var out UserProfile
	err := c.doJSON(ctx, http.MethodGet, "/api/users/me", nil, &out)
	return out, err
}

func (c *Client) SearchUsers(ctx context.Context, username string, limit int) ([]UserProfile, error) {
	query := url.Values{}
	query.Set("username", username)
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	var out []UserProfile
	err := c.doJSON(ctx, http.MethodGet, "/api/users?"+query.Encode(), nil, &out)
	return out, err
}

func (c *Client) ListConversations(ctx context.Context, limit int) ([]ConversationSummary, error) {
	path := "/api/conversations"
	if limit > 0 {
		values := url.Values{}
		values.Set("limit", strconv.Itoa(limit))
		path += "?" + values.Encode()
	}
	var out []ConversationSummary
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) OpenConversation(ctx context.Context, username string) (ConversationDetails, error) {
	var out ConversationDetails
	err := c.doJSON(ctx, http.MethodPost, "/api/conversations", map[string]string{"username": username}, &out)
	return out, err
}

func (c *Client) GetConversation(ctx context.Context, conversationID string) (ConversationDetails, error) {
	var out ConversationDetails
	err := c.doJSON(ctx, http.MethodGet, "/api/conversations/"+url.PathEscape(conversationID), nil, &out)
	return out, err
}

func (c *Client) ListMessages(ctx context.Context, conversationID string, before string, limit int) ([]MessageDetails, error) {
	values := url.Values{}
	if strings.TrimSpace(before) != "" {
		values.Set("before", before)
	}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}

	requestPath := "/api/conversations/" + url.PathEscape(conversationID) + "/messages"
	if len(values) > 0 {
		requestPath += "?" + values.Encode()
	}

	var out []MessageDetails
	err := c.doJSON(ctx, http.MethodGet, requestPath, nil, &out)
	return out, err
}

type SendMessageRequest struct {
	Content        *string         `json:"content,omitempty"`
	Kind           string          `json:"kind,omitempty"`
	JSONRenderSpec json.RawMessage `json:"json_render_spec,omitempty"`
}

func (c *Client) SendMessage(ctx context.Context, conversationID string, input SendMessageRequest) (Message, error) {
	var out Message
	err := c.doJSON(ctx, http.MethodPost, "/api/conversations/"+url.PathEscape(conversationID)+"/messages", input, &out)
	return out, err
}

func (c *Client) EditMessage(ctx context.Context, messageID, content string) (Message, error) {
	var out Message
	err := c.doJSON(ctx, http.MethodPatch, "/api/messages/"+url.PathEscape(messageID), map[string]string{
		"content": content,
	}, &out)
	return out, err
}

func (c *Client) DeleteMessage(ctx context.Context, messageID string) (Message, error) {
	var out Message
	err := c.doJSON(ctx, http.MethodDelete, "/api/messages/"+url.PathEscape(messageID), nil, &out)
	return out, err
}

func (c *Client) AddReaction(ctx context.Context, messageID, emoji string) (ToggleReactionResult, error) {
	var out ToggleReactionResult
	err := c.doJSON(ctx, http.MethodPost, "/api/messages/"+url.PathEscape(messageID)+"/reactions", map[string]string{
		"emoji": emoji,
	}, &out)
	return out, err
}

func (c *Client) RemoveReaction(ctx context.Context, messageID, emoji string) (Reaction, error) {
	var out Reaction
	err := c.doJSON(ctx, http.MethodDelete, "/api/messages/"+url.PathEscape(messageID)+"/reactions/"+url.PathEscape(emoji), nil, &out)
	return out, err
}

func (c *Client) EventStreamURL() (string, error) {
	if c.baseURL == nil {
		return "", errors.New("server URL is not configured")
	}

	streamURL := *c.baseURL
	joined := path.Join(streamURL.Path, "api", "events")
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	streamURL.Path = joined
	query := streamURL.Query()
	if c.token != "" {
		query.Set("token", c.token)
	}
	streamURL.RawQuery = query.Encode()
	return streamURL.String(), nil
}

func (c *Client) doJSON(ctx context.Context, method, requestPath string, in, out any) error {
	if c.baseURL == nil {
		return errors.New("server URL is not configured")
	}
	parsedRequestPath, err := url.Parse(requestPath)
	if err != nil {
		return fmt.Errorf("parse request path: %w", err)
	}
	if parsedRequestPath.IsAbs() {
		return errors.New("request path must be relative")
	}

	var body io.Reader
	if in != nil {
		encoded, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	u := *c.baseURL
	if strings.HasPrefix(parsedRequestPath.Path, "/") {
		u.Path = strings.TrimSuffix(c.baseURL.Path, "/") + parsedRequestPath.Path
	} else {
		u.Path = strings.TrimSuffix(c.baseURL.Path, "/") + "/" + parsedRequestPath.Path
	}
	u.RawQuery = parsedRequestPath.RawQuery
	u.Fragment = parsedRequestPath.Fragment

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var payload errorPayload
		if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
			payload.Error = strings.TrimSpace(resp.Status)
		}
		return &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(payload.Error)}
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}
	return nil
}

// UserProfile is the safe user projection from API responses.
type UserProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthResponse matches POST /api/auth/register and /api/auth/login.
type AuthResponse struct {
	Token string      `json:"token"`
	User  UserProfile `json:"user"`
}

// Conversation matches the conversation model shape.
type Conversation struct {
	ID           string    `json:"id"`
	ParticipantA string    `json:"participant_a"`
	ParticipantB string    `json:"participant_b"`
	CreatedAt    time.Time `json:"created_at"`
}

// Message matches the message model shape.
type Message struct {
	ID             string          `json:"id"`
	ConversationID string          `json:"conversation_id"`
	SenderID       string          `json:"sender_id"`
	Content        *string         `json:"content,omitempty"`
	Kind           string          `json:"kind,omitempty"`
	JSONRenderSpec json.RawMessage `json:"json_render_spec,omitempty"`
	AttachmentURL  *string         `json:"attachment_url,omitempty"`
	AttachmentType *string         `json:"attachment_type,omitempty"`
	Edited         bool            `json:"edited"`
	Deleted        bool            `json:"deleted"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Reaction matches the reaction model shape.
type Reaction struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Emoji     string    `json:"emoji"`
	CreatedAt time.Time `json:"created_at"`
}

// ToggleReactionResult is returned by POST /api/messages/:id/reactions.
type ToggleReactionResult struct {
	Action   string   `json:"action"`
	Reaction Reaction `json:"reaction"`
}

// ConversationSummary is returned by GET /api/conversations.
type ConversationSummary struct {
	Conversation Conversation `json:"conversation"`
	OtherUser    UserProfile  `json:"other_user"`
	LastMessage  *Message     `json:"last_message,omitempty"`
}

// ConversationDetails is returned by GET/POST /api/conversations/:id.
type ConversationDetails struct {
	Conversation Conversation `json:"conversation"`
	ParticipantA UserProfile  `json:"participant_a"`
	ParticipantB UserProfile  `json:"participant_b"`
}

// MessageDetails is returned by GET /api/conversations/:id/messages.
type MessageDetails struct {
	Message Message     `json:"message"`
	Sender  UserProfile `json:"sender"`
}

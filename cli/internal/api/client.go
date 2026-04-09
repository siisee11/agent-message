package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"path/filepath"
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

// Client is a REST client for the Agent Message server contract.
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
	raw, err := validateServerURLInput(serverURL)
	if err != nil {
		return err
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
	if parsed.RawQuery != "" {
		return errors.New("server URL must not contain a query string")
	}
	if parsed.Fragment != "" {
		return errors.New("server URL must not contain a fragment")
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

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *Client) Register(ctx context.Context, username, password string) (AuthResponse, error) {
	return c.RegisterWithRequest(ctx, AuthRequest{
		Username: username,
		Password: password,
	})
}

func (c *Client) RegisterWithRequest(ctx context.Context, input AuthRequest) (AuthResponse, error) {
	normalizedUsername, err := validateUsername(input.Username)
	if err != nil {
		return AuthResponse{}, err
	}

	var out AuthResponse
	err = c.doJSON(ctx, http.MethodPost, "/api/auth/register", map[string]string{
		"username": normalizedUsername,
		"password": input.Password,
	}, &out)
	return out, err
}

func (c *Client) Login(ctx context.Context, username, password string) (AuthResponse, error) {
	return c.LoginWithRequest(ctx, AuthRequest{
		Username: username,
		Password: password,
	})
}

func (c *Client) LoginWithRequest(ctx context.Context, input AuthRequest) (AuthResponse, error) {
	normalizedUsername, err := validateUsername(input.Username)
	if err != nil {
		return AuthResponse{}, err
	}

	var out AuthResponse
	err = c.doJSON(ctx, http.MethodPost, "/api/auth/login", map[string]string{
		"username": normalizedUsername,
		"password": input.Password,
	}, &out)
	return out, err
}

func (c *Client) Logout(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/auth/logout", nil, nil)
}

func (c *Client) GetCatalogPrompt(ctx context.Context) (CatalogPromptResponse, error) {
	var out CatalogPromptResponse
	err := c.doJSON(ctx, http.MethodGet, "/api/catalog/prompt", nil, &out)
	return out, err
}

func (c *Client) Me(ctx context.Context) (UserProfile, error) {
	var out UserProfile
	err := c.doJSON(ctx, http.MethodGet, "/api/users/me", nil, &out)
	return out, err
}

func (c *Client) SearchUsers(ctx context.Context, username string, limit int) ([]UserProfile, error) {
	normalizedUsername, err := validateUsername(username)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("username", normalizedUsername)
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	var out []UserProfile
	err = c.doJSON(ctx, http.MethodGet, "/api/users?"+query.Encode(), nil, &out)
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

type OpenConversationRequest struct {
	Username string `json:"username"`
}

func (c *Client) OpenConversation(ctx context.Context, username string) (ConversationDetails, error) {
	return c.OpenConversationWithRequest(ctx, OpenConversationRequest{
		Username: username,
	})
}

func (c *Client) OpenConversationWithRequest(ctx context.Context, input OpenConversationRequest) (ConversationDetails, error) {
	normalizedUsername, err := validateUsername(input.Username)
	if err != nil {
		return ConversationDetails{}, err
	}

	var out ConversationDetails
	err = c.doJSON(ctx, http.MethodPost, "/api/conversations", map[string]string{"username": normalizedUsername}, &out)
	return out, err
}

func (c *Client) GetConversation(ctx context.Context, conversationID string) (ConversationDetails, error) {
	normalizedConversationID, err := validateResourceID("conversation ID", conversationID)
	if err != nil {
		return ConversationDetails{}, err
	}

	var out ConversationDetails
	err = c.doJSON(ctx, http.MethodGet, "/api/conversations/"+url.PathEscape(normalizedConversationID), nil, &out)
	return out, err
}

func (c *Client) ListMessages(ctx context.Context, conversationID string, before string, limit int) ([]MessageDetails, error) {
	normalizedConversationID, err := validateResourceID("conversation ID", conversationID)
	if err != nil {
		return nil, err
	}

	values := url.Values{}
	if strings.TrimSpace(before) != "" {
		normalizedBefore, err := validateResourceID("before cursor", before)
		if err != nil {
			return nil, err
		}
		values.Set("before", normalizedBefore)
	}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}

	requestPath := "/api/conversations/" + url.PathEscape(normalizedConversationID) + "/messages"
	if len(values) > 0 {
		requestPath += "?" + values.Encode()
	}

	var out []MessageDetails
	err = c.doJSON(ctx, http.MethodGet, requestPath, nil, &out)
	return out, err
}

type SendMessageRequest struct {
	Content        *string         `json:"content,omitempty"`
	Kind           string          `json:"kind,omitempty"`
	JSONRenderSpec json.RawMessage `json:"json_render_spec,omitempty"`
}

type SendAttachmentMessageRequest struct {
	Content        *string
	AttachmentPath string
}

func (c *Client) SendMessage(ctx context.Context, conversationID string, input SendMessageRequest) (Message, error) {
	normalizedConversationID, err := validateResourceID("conversation ID", conversationID)
	if err != nil {
		return Message{}, err
	}

	var out Message
	err = c.doJSON(ctx, http.MethodPost, "/api/conversations/"+url.PathEscape(normalizedConversationID)+"/messages", input, &out)
	return out, err
}

func (c *Client) SendAttachmentMessage(ctx context.Context, conversationID string, input SendAttachmentMessageRequest) (Message, error) {
	if c.baseURL == nil {
		return Message{}, errors.New("server URL is not configured")
	}

	normalizedConversationID, err := validateResourceID("conversation ID", conversationID)
	if err != nil {
		return Message{}, err
	}

	attachmentPath, err := validateAttachmentPath(input.AttachmentPath)
	if err != nil {
		return Message{}, err
	}

	file, err := os.Open(attachmentPath)
	if err != nil {
		return Message{}, fmt.Errorf("open attachment: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return Message{}, fmt.Errorf("stat attachment: %w", err)
	}
	if info.IsDir() {
		return Message{}, errors.New("attachment path must be a file")
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if input.Content != nil {
		trimmedContent := strings.TrimSpace(*input.Content)
		if trimmedContent != "" {
			if err := writer.WriteField("content", trimmedContent); err != nil {
				return Message{}, fmt.Errorf("write multipart content field: %w", err)
			}
		}
	}

	part, err := writer.CreatePart(buildAttachmentPartHeader(filepath.Base(attachmentPath), detectAttachmentContentType(file, attachmentPath)))
	if err != nil {
		return Message{}, fmt.Errorf("create multipart attachment field: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return Message{}, fmt.Errorf("write multipart attachment field: %w", err)
	}
	if err := writer.Close(); err != nil {
		return Message{}, fmt.Errorf("close multipart writer: %w", err)
	}

	u := *c.baseURL
	u.Path = strings.TrimSuffix(c.baseURL.Path, "/") + "/api/conversations/" + url.PathEscape(normalizedConversationID) + "/messages"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), &body)
	if err != nil {
		return Message{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var payload errorPayload
		if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
			payload.Error = strings.TrimSpace(resp.Status)
		}
		return Message{}, &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(payload.Error)}
	}

	var out Message
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Message{}, fmt.Errorf("decode response body: %w", err)
	}
	return out, nil
}

func buildAttachmentPartHeader(filename, contentType string) textproto.MIMEHeader {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="attachment"; filename="%s"`, escapeMultipartFilename(filename)))
	header.Set("Content-Type", contentType)
	return header
}

func escapeMultipartFilename(filename string) string {
	replacer := strings.NewReplacer("\\", "\\\\", `"`, "\\\"")
	return replacer.Replace(filename)
}

func detectAttachmentContentType(file *os.File, attachmentPath string) string {
	contentType := strings.TrimSpace(mime.TypeByExtension(strings.ToLower(filepath.Ext(attachmentPath))))
	if contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err == nil && strings.TrimSpace(mediaType) != "" {
			return mediaType
		}
	}

	var sniff [512]byte
	n, err := file.Read(sniff[:])
	if err == nil || errors.Is(err, io.EOF) {
		if _, seekErr := file.Seek(0, io.SeekStart); seekErr == nil && n > 0 {
			return http.DetectContentType(sniff[:n])
		}
	}
	if _, seekErr := file.Seek(0, io.SeekStart); seekErr == nil {
		return "application/octet-stream"
	}
	return "application/octet-stream"
}

type EditMessageRequest struct {
	Content string `json:"content"`
}

func (c *Client) EditMessage(ctx context.Context, messageID, content string) (Message, error) {
	return c.EditMessageWithRequest(ctx, messageID, EditMessageRequest{
		Content: content,
	})
}

func (c *Client) EditMessageWithRequest(ctx context.Context, messageID string, input EditMessageRequest) (Message, error) {
	normalizedMessageID, err := validateResourceID("message ID", messageID)
	if err != nil {
		return Message{}, err
	}

	var out Message
	err = c.doJSON(ctx, http.MethodPatch, "/api/messages/"+url.PathEscape(normalizedMessageID), map[string]string{
		"content": input.Content,
	}, &out)
	return out, err
}

func (c *Client) DeleteMessage(ctx context.Context, messageID string) (Message, error) {
	normalizedMessageID, err := validateResourceID("message ID", messageID)
	if err != nil {
		return Message{}, err
	}

	var out Message
	err = c.doJSON(ctx, http.MethodDelete, "/api/messages/"+url.PathEscape(normalizedMessageID), nil, &out)
	return out, err
}

type ToggleReactionRequest struct {
	Emoji string `json:"emoji"`
}

func (c *Client) AddReaction(ctx context.Context, messageID, emoji string) (ToggleReactionResult, error) {
	return c.AddReactionWithRequest(ctx, messageID, ToggleReactionRequest{
		Emoji: emoji,
	})
}

func (c *Client) AddReactionWithRequest(ctx context.Context, messageID string, input ToggleReactionRequest) (ToggleReactionResult, error) {
	normalizedMessageID, err := validateResourceID("message ID", messageID)
	if err != nil {
		return ToggleReactionResult{}, err
	}
	if hasControlCharacters(input.Emoji) {
		return ToggleReactionResult{}, errors.New("emoji must not contain control characters")
	}

	var out ToggleReactionResult
	err = c.doJSON(ctx, http.MethodPost, "/api/messages/"+url.PathEscape(normalizedMessageID)+"/reactions", map[string]string{
		"emoji": input.Emoji,
	}, &out)
	return out, err
}

func (c *Client) RemoveReaction(ctx context.Context, messageID, emoji string) (Reaction, error) {
	normalizedMessageID, err := validateResourceID("message ID", messageID)
	if err != nil {
		return Reaction{}, err
	}
	if hasControlCharacters(emoji) {
		return Reaction{}, errors.New("emoji must not contain control characters")
	}

	var out Reaction
	err = c.doJSON(ctx, http.MethodDelete, "/api/messages/"+url.PathEscape(normalizedMessageID)+"/reactions/"+url.PathEscape(emoji), nil, &out)
	return out, err
}

func (c *Client) EventStreamURL(clientKind string) (string, error) {
	return c.EventStreamURLWithWatcherSession(clientKind, "")
}

func (c *Client) EventStreamURLWithWatcherSession(clientKind, watcherSessionID string) (string, error) {
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
	if strings.TrimSpace(clientKind) != "" {
		query.Set("client_kind", strings.TrimSpace(clientKind))
	}
	if strings.TrimSpace(watcherSessionID) != "" {
		query.Set("watcher_session_id", strings.TrimSpace(watcherSessionID))
	}
	streamURL.RawQuery = query.Encode()
	return streamURL.String(), nil
}

type WatcherHeartbeatRequest struct {
	SessionID string `json:"session_id"`
}

func (c *Client) WatcherHeartbeat(ctx context.Context, sessionID string) error {
	normalizedSessionID, err := validateWatcherSessionID(sessionID)
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodPost, "/api/watchers/heartbeat", WatcherHeartbeatRequest{
		SessionID: normalizedSessionID,
	}, nil)
}

func (c *Client) UnregisterWatcherSession(ctx context.Context, sessionID string) error {
	normalizedSessionID, err := validateWatcherSessionID(sessionID)
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodDelete, "/api/watchers/sessions/"+url.PathEscape(normalizedSessionID), nil, nil)
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
	if parsedRequestPath.Fragment != "" {
		return errors.New("request path must not contain a fragment")
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

// CatalogPromptResponse matches GET /api/catalog/prompt.
type CatalogPromptResponse struct {
	Prompt string `json:"prompt"`
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
	Conversation    Conversation `json:"conversation"`
	OtherUser       UserProfile  `json:"other_user"`
	LastMessage     *Message     `json:"last_message,omitempty"`
	SessionFolder   string       `json:"session_folder,omitempty"`
	SessionHostname string       `json:"session_hostname,omitempty"`
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

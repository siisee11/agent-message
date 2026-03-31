package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultDirName   = ".agent-messenger"
	defaultFileName  = "config"
	defaultServerURL = "http://localhost:8080"
)

// Profile stores auth and read-session state for a saved account.
type Profile struct {
	Username               string                 `json:"username"`
	ServerURL              string                 `json:"server_url"`
	Token                  string                 `json:"token,omitempty"`
	LastReadConversationID string                 `json:"last_read_conversation_id,omitempty"`
	ReadSessions           map[string]ReadSession `json:"read_sessions,omitempty"`
}

// ReadSession stores index -> message ID data from the latest `read` per conversation.
type ReadSession struct {
	ConversationID  string         `json:"conversation_id"`
	Username        string         `json:"username,omitempty"`
	IndexToMessage  map[int]string `json:"index_to_message,omitempty"`
	LastReadMessage string         `json:"last_read_message,omitempty"`
}

// Config is persisted at ~/.agent-messenger/config.
type Config struct {
	ServerURL              string                 `json:"server_url"`
	Token                  string                 `json:"token,omitempty"`
	ActiveProfile          string                 `json:"active_profile,omitempty"`
	Profiles               map[string]Profile     `json:"profiles,omitempty"`
	LastReadConversationID string                 `json:"last_read_conversation_id,omitempty"`
	ReadSessions           map[string]ReadSession `json:"read_sessions,omitempty"`
}

// Store provides disk persistence for Config.
type Store struct {
	path string
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(defaultDirName, defaultFileName)
	}
	return filepath.Join(home, defaultDirName, defaultFileName)
}

func DefaultServerURL() string {
	return defaultServerURL
}

func NewStore(path string) *Store {
	p := strings.TrimSpace(path)
	if p == "" {
		p = DefaultPath()
	}
	return &Store{path: p}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config %q: %w", s.path, err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", s.path, err)
	}

	if err := cfg.normalizeLoaded(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (s *Store) Save(cfg Config) error {
	if err := cfg.prepareForSave(); err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory %q: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp config %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace config %q: %w", s.path, err)
	}
	return nil
}

func defaultConfig() Config {
	return Config{
		ServerURL:    defaultServerURL,
		Profiles:     make(map[string]Profile),
		ReadSessions: make(map[string]ReadSession),
	}
}

func (c *Config) prepareForSave() error {
	if c == nil {
		return errors.New("config is required")
	}

	c.ActiveProfile = strings.TrimSpace(c.ActiveProfile)
	if c.Profiles == nil {
		c.Profiles = make(map[string]Profile)
	}
	if c.ActiveProfile != "" {
		c.Profiles[c.ActiveProfile] = Profile{
			Username:               c.ActiveProfile,
			ServerURL:              c.ServerURL,
			Token:                  c.Token,
			LastReadConversationID: c.LastReadConversationID,
			ReadSessions:           cloneReadSessions(c.ReadSessions),
		}
	}

	return c.normalizeLoaded()
}

func (c *Config) normalizeLoaded() error {
	if c == nil {
		return errors.New("config is required")
	}

	normalizedServerURL, err := normalizeServerURL(c.ServerURL)
	if err != nil {
		return err
	}
	c.ServerURL = normalizedServerURL
	c.Token = strings.TrimSpace(c.Token)
	c.ActiveProfile = strings.TrimSpace(c.ActiveProfile)
	c.LastReadConversationID = strings.TrimSpace(c.LastReadConversationID)
	c.ReadSessions = normalizeReadSessions(c.ReadSessions)
	c.LastReadConversationID = normalizeLastReadConversationID(c.LastReadConversationID, c.ReadSessions)

	if c.Profiles == nil {
		c.Profiles = make(map[string]Profile)
	}
	normalizedProfiles := make(map[string]Profile, len(c.Profiles))
	for key, profile := range c.Profiles {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}

		normalizedProfile, err := normalizeProfile(normalizedKey, profile)
		if err != nil {
			return err
		}
		normalizedProfiles[normalizedKey] = normalizedProfile
	}
	c.Profiles = normalizedProfiles

	if c.ActiveProfile == "" {
		return nil
	}

	profile, ok := c.Profiles[c.ActiveProfile]
	if !ok {
		c.ActiveProfile = ""
		return nil
	}

	c.ServerURL = profile.ServerURL
	c.Token = profile.Token
	c.ReadSessions = cloneReadSessions(profile.ReadSessions)
	c.LastReadConversationID = normalizeLastReadConversationID(profile.LastReadConversationID, c.ReadSessions)
	return nil
}

func normalizeProfile(name string, profile Profile) (Profile, error) {
	serverURL, err := normalizeServerURL(profile.ServerURL)
	if err != nil {
		return Profile{}, fmt.Errorf("invalid profile %q: %w", name, err)
	}

	profile.Username = strings.TrimSpace(profile.Username)
	if profile.Username == "" {
		profile.Username = name
	}
	profile.ServerURL = serverURL
	profile.Token = strings.TrimSpace(profile.Token)
	profile.ReadSessions = normalizeReadSessions(profile.ReadSessions)
	profile.LastReadConversationID = normalizeLastReadConversationID(profile.LastReadConversationID, profile.ReadSessions)
	return profile, nil
}

func normalizeServerURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = defaultServerURL
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid server_url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("server_url must start with http:// or https://")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", errors.New("server_url host is required")
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	parsed.RawPath = strings.TrimSuffix(parsed.RawPath, "/")
	return parsed.String(), nil
}

func normalizeReadSessions(readSessions map[string]ReadSession) map[string]ReadSession {
	if readSessions == nil {
		return make(map[string]ReadSession)
	}

	normalizedSessions := make(map[string]ReadSession, len(readSessions))
	for key, session := range readSessions {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}

		session.ConversationID = strings.TrimSpace(session.ConversationID)
		session.Username = strings.TrimSpace(session.Username)
		session.LastReadMessage = strings.TrimSpace(session.LastReadMessage)
		if session.IndexToMessage == nil {
			session.IndexToMessage = make(map[int]string)
		}
		if session.ConversationID == "" {
			session.ConversationID = normalizedKey
		}
		normalizedSessions[normalizedKey] = session
	}
	return normalizedSessions
}

func normalizeLastReadConversationID(lastReadConversationID string, sessions map[string]ReadSession) string {
	normalized := strings.TrimSpace(lastReadConversationID)
	if normalized == "" {
		return ""
	}
	if _, ok := sessions[normalized]; !ok {
		return ""
	}
	return normalized
}

func cloneReadSessions(readSessions map[string]ReadSession) map[string]ReadSession {
	cloned := make(map[string]ReadSession, len(readSessions))
	for key, session := range readSessions {
		indexToMessage := make(map[int]string, len(session.IndexToMessage))
		for index, messageID := range session.IndexToMessage {
			indexToMessage[index] = messageID
		}
		cloned[key] = ReadSession{
			ConversationID:  session.ConversationID,
			Username:        session.Username,
			IndexToMessage:  indexToMessage,
			LastReadMessage: session.LastReadMessage,
		}
	}
	return cloned
}

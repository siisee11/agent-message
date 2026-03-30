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
	defaultDirName   = ".msgr"
	defaultFileName  = "config"
	defaultServerURL = "http://localhost:8080"
)

// ReadSession stores index -> message ID data from the latest `read` per conversation.
type ReadSession struct {
	ConversationID  string         `json:"conversation_id"`
	Username        string         `json:"username,omitempty"`
	IndexToMessage  map[int]string `json:"index_to_message,omitempty"`
	LastReadMessage string         `json:"last_read_message,omitempty"`
}

// Config is persisted at ~/.msgr/config.
type Config struct {
	ServerURL    string                 `json:"server_url"`
	Token        string                 `json:"token,omitempty"`
	ReadSessions map[string]ReadSession `json:"read_sessions,omitempty"`
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
	cfg := Config{
		ServerURL:    defaultServerURL,
		ReadSessions: make(map[string]ReadSession),
	}

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

	if err := cfg.normalize(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (s *Store) Save(cfg Config) error {
	if err := cfg.normalize(); err != nil {
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

func (c *Config) normalize() error {
	trimmedToken := strings.TrimSpace(c.Token)
	c.Token = trimmedToken

	rawURL := strings.TrimSpace(c.ServerURL)
	if rawURL == "" {
		rawURL = defaultServerURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid server_url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("server_url must start with http:// or https://")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return errors.New("server_url host is required")
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	parsed.RawPath = strings.TrimSuffix(parsed.RawPath, "/")
	c.ServerURL = parsed.String()

	if c.ReadSessions == nil {
		c.ReadSessions = make(map[string]ReadSession)
	}
	for key, session := range c.ReadSessions {
		session.ConversationID = strings.TrimSpace(session.ConversationID)
		session.Username = strings.TrimSpace(session.Username)
		session.LastReadMessage = strings.TrimSpace(session.LastReadMessage)
		if session.IndexToMessage == nil {
			session.IndexToMessage = make(map[int]string)
		}
		c.ReadSessions[strings.TrimSpace(key)] = session
	}

	return nil
}

package realtime

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

const DefaultWatcherPresenceTTL = 30 * time.Second

var (
	ErrWatcherPresenceUserIDRequired    = errors.New("watcher presence user id is required")
	ErrWatcherPresenceSessionIDRequired = errors.New("watcher presence session id is required")
)

type WatcherPresenceTransition struct {
	UserID          string
	ConversationIDs []string
	Online          bool
}

type WatcherPresence struct {
	mu           sync.RWMutex
	sessions     map[string]watcherPresenceSession
	userSessions map[string]map[string]struct{}
	ttl          time.Duration
	nowFn        func() time.Time
}

type watcherPresenceSession struct {
	userID        string
	sessionID     string
	conversations map[string]struct{}
	expiresAt     time.Time
}

func NewWatcherPresence(ttl time.Duration) *WatcherPresence {
	if ttl <= 0 {
		ttl = DefaultWatcherPresenceTTL
	}
	return &WatcherPresence{
		sessions:     make(map[string]watcherPresenceSession),
		userSessions: make(map[string]map[string]struct{}),
		ttl:          ttl,
		nowFn:        time.Now,
	}
}

func (p *WatcherPresence) SetNowFnForTests(nowFn func() time.Time) {
	if nowFn == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nowFn = nowFn
}

func (p *WatcherPresence) Register(userID, sessionID string, conversationIDs []string) (*WatcherPresenceTransition, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrWatcherPresenceUserIDRequired
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, ErrWatcherPresenceSessionIDRequired
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := p.nowFn().UTC()
	beforeOnline := p.hasActiveSessionsLocked(userID, now)

	session := watcherPresenceSession{
		userID:        userID,
		sessionID:     sessionID,
		conversations: normalizeConversationSet(conversationIDs),
		expiresAt:     now.Add(p.ttl),
	}
	p.sessions[watcherPresenceKey(userID, sessionID)] = session
	if _, ok := p.userSessions[userID]; !ok {
		p.userSessions[userID] = make(map[string]struct{})
	}
	p.userSessions[userID][sessionID] = struct{}{}

	if beforeOnline {
		return nil, nil
	}
	return &WatcherPresenceTransition{
		UserID:          userID,
		ConversationIDs: p.userConversationIDsLocked(userID),
		Online:          true,
	}, nil
}

func (p *WatcherPresence) Heartbeat(userID, sessionID string) (*WatcherPresenceTransition, bool) {
	userID = strings.TrimSpace(userID)
	sessionID = strings.TrimSpace(sessionID)
	if userID == "" || sessionID == "" {
		return nil, false
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := watcherPresenceKey(userID, sessionID)
	session, ok := p.sessions[key]
	if !ok {
		return nil, false
	}

	now := p.nowFn().UTC()
	beforeOnline := p.hasActiveSessionsLocked(userID, now)
	session.expiresAt = now.Add(p.ttl)
	p.sessions[key] = session

	if beforeOnline {
		return nil, true
	}
	return &WatcherPresenceTransition{
		UserID:          userID,
		ConversationIDs: p.userConversationIDsLocked(userID),
		Online:          true,
	}, true
}

func (p *WatcherPresence) Unregister(userID, sessionID string) *WatcherPresenceTransition {
	userID = strings.TrimSpace(userID)
	sessionID = strings.TrimSpace(sessionID)
	if userID == "" || sessionID == "" {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	key := watcherPresenceKey(userID, sessionID)
	if _, ok := p.sessions[key]; !ok {
		return nil
	}

	now := p.nowFn().UTC()
	beforeOnline := p.hasActiveSessionsLocked(userID, now)
	beforeConversationIDs := p.userConversationIDsLocked(userID)

	delete(p.sessions, key)
	p.removeUserSessionLocked(userID, sessionID)

	if !beforeOnline || p.hasActiveSessionsLocked(userID, now) {
		return nil
	}
	return &WatcherPresenceTransition{
		UserID:          userID,
		ConversationIDs: beforeConversationIDs,
		Online:          false,
	}
}

func (p *WatcherPresence) SubscribeUser(userID, conversationID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrWatcherPresenceUserIDRequired
	}
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return ErrConversationIDMissing
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	for sessionID := range p.userSessions[userID] {
		key := watcherPresenceKey(userID, sessionID)
		session, ok := p.sessions[key]
		if !ok {
			continue
		}
		session.conversations[conversationID] = struct{}{}
		p.sessions[key] = session
	}
	return nil
}

func (p *WatcherPresence) IsOnline(userID string) bool {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.hasActiveSessionsLocked(userID, p.nowFn().UTC())
}

func (p *WatcherPresence) Expire() []WatcherPresenceTransition {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := p.nowFn().UTC()
	usersWithExpiredSessions := make(map[string]struct{})
	for _, session := range p.sessions {
		if session.expiresAt.After(now) {
			continue
		}
		usersWithExpiredSessions[session.userID] = struct{}{}
	}

	transitions := make([]WatcherPresenceTransition, 0, len(usersWithExpiredSessions))
	for userID := range usersWithExpiredSessions {
		beforeConversationIDs := p.userConversationIDsLocked(userID)
		removedAny := false
		for sessionID := range p.userSessions[userID] {
			key := watcherPresenceKey(userID, sessionID)
			session, ok := p.sessions[key]
			if !ok || session.expiresAt.After(now) {
				continue
			}
			delete(p.sessions, key)
			p.removeUserSessionLocked(userID, sessionID)
			removedAny = true
		}
		if !removedAny || p.hasActiveSessionsLocked(userID, now) {
			continue
		}
		transitions = append(transitions, WatcherPresenceTransition{
			UserID:          userID,
			ConversationIDs: beforeConversationIDs,
			Online:          false,
		})
	}
	return transitions
}

func (p *WatcherPresence) hasActiveSessionsLocked(userID string, now time.Time) bool {
	for sessionID := range p.userSessions[userID] {
		session, ok := p.sessions[watcherPresenceKey(userID, sessionID)]
		if !ok {
			continue
		}
		if session.expiresAt.After(now) {
			return true
		}
	}
	return false
}

func (p *WatcherPresence) userConversationIDsLocked(userID string) []string {
	conversationSet := make(map[string]struct{})
	for sessionID := range p.userSessions[userID] {
		session, ok := p.sessions[watcherPresenceKey(userID, sessionID)]
		if !ok {
			continue
		}
		for conversationID := range session.conversations {
			conversationSet[conversationID] = struct{}{}
		}
	}

	out := make([]string, 0, len(conversationSet))
	for conversationID := range conversationSet {
		out = append(out, conversationID)
	}
	sort.Strings(out)
	return out
}

func (p *WatcherPresence) removeUserSessionLocked(userID, sessionID string) {
	sessions, ok := p.userSessions[userID]
	if !ok {
		return
	}
	delete(sessions, sessionID)
	if len(sessions) == 0 {
		delete(p.userSessions, userID)
	}
}

func watcherPresenceKey(userID, sessionID string) string {
	return userID + "\x00" + sessionID
}

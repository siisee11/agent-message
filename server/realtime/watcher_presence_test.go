package realtime

import (
	"reflect"
	"testing"
	"time"
)

func TestWatcherPresenceRegisterHeartbeatAndExpire(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	presence := NewWatcherPresence(30 * time.Second)
	presence.SetNowFnForTests(func() time.Time { return now })

	transition, err := presence.Register("u1", "session-1", []string{"conv-a"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if transition == nil || !transition.Online {
		t.Fatalf("expected online transition, got %+v", transition)
	}
	if !reflect.DeepEqual(transition.ConversationIDs, []string{"conv-a"}) {
		t.Fatalf("unexpected online transition conversations: %+v", transition.ConversationIDs)
	}
	if !presence.IsOnline("u1") {
		t.Fatalf("expected user to be online after register")
	}

	now = now.Add(10 * time.Second)
	if transition, ok := presence.Heartbeat("u1", "session-1"); !ok {
		t.Fatalf("expected heartbeat to succeed")
	} else if transition != nil {
		t.Fatalf("expected no heartbeat transition while still online, got %+v", transition)
	}

	now = now.Add(31 * time.Second)
	if presence.IsOnline("u1") {
		t.Fatalf("expected user to be offline after lease expiry")
	}

	transitions := presence.Expire()
	if len(transitions) != 1 {
		t.Fatalf("expected one expiry transition, got %+v", transitions)
	}
	if transitions[0].Online {
		t.Fatalf("expected offline expiry transition, got %+v", transitions[0])
	}
	if !reflect.DeepEqual(transitions[0].ConversationIDs, []string{"conv-a"}) {
		t.Fatalf("unexpected expiry transition conversations: %+v", transitions[0].ConversationIDs)
	}
}

func TestWatcherPresenceSubscribeUserUpdatesTrackedConversations(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	presence := NewWatcherPresence(30 * time.Second)
	presence.SetNowFnForTests(func() time.Time { return now })

	if _, err := presence.Register("u1", "session-1", []string{"conv-a"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := presence.SubscribeUser("u1", "conv-b"); err != nil {
		t.Fatalf("subscribe user conversation: %v", err)
	}

	transition := presence.Unregister("u1", "session-1")
	if transition == nil || transition.Online {
		t.Fatalf("expected offline transition on unregister, got %+v", transition)
	}
	if !reflect.DeepEqual(transition.ConversationIDs, []string{"conv-a", "conv-b"}) {
		t.Fatalf("unexpected offline conversations: %+v", transition.ConversationIDs)
	}
}

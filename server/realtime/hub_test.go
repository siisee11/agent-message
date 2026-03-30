package realtime

import (
	"errors"
	"reflect"
	"testing"
)

func TestHubRegisterBroadcastAndUnregister(t *testing.T) {
	hub := NewHub()

	clientA := &Client{UserID: "u1", Send: make(chan Event, 2)}
	clientB := &Client{UserID: "u2", Send: make(chan Event, 2)}
	clientC := &Client{UserID: "u3", Send: make(chan Event, 2)}

	if err := hub.Register(clientA, []string{"conv-a", "conv-b"}); err != nil {
		t.Fatalf("register clientA: %v", err)
	}
	if err := hub.Register(clientB, []string{"conv-a"}); err != nil {
		t.Fatalf("register clientB: %v", err)
	}
	if err := hub.Register(clientC, []string{"conv-b"}); err != nil {
		t.Fatalf("register clientC: %v", err)
	}

	if got := hub.ConnectionCount(); got != 3 {
		t.Fatalf("expected 3 connected clients, got %d", got)
	}
	if got := hub.ConnectionsForUser("u1"); got != 1 {
		t.Fatalf("expected 1 connection for u1, got %d", got)
	}

	event := Event{Type: EventTypeMessageNew, Data: map[string]any{"id": "m1"}}
	result, err := hub.BroadcastToConversation("conv-a", event)
	if err != nil {
		t.Fatalf("broadcast to conv-a: %v", err)
	}
	if result.Delivered != 2 || result.Dropped != 0 {
		t.Fatalf("unexpected broadcast result: %+v", result)
	}

	assertEvent(t, clientA.Send, EventTypeMessageNew)
	assertEvent(t, clientB.Send, EventTypeMessageNew)
	assertNoEvent(t, clientC.Send)

	hub.Unregister(clientB)
	if got := hub.ConnectionCount(); got != 2 {
		t.Fatalf("expected 2 connected clients after unregister, got %d", got)
	}
	if got := hub.ConnectionsForUser("u2"); got != 0 {
		t.Fatalf("expected 0 connections for u2 after unregister, got %d", got)
	}

	result, err = hub.BroadcastToConversation("conv-a", event)
	if err != nil {
		t.Fatalf("broadcast to conv-a after unregister: %v", err)
	}
	if result.Delivered != 1 || result.Dropped != 0 {
		t.Fatalf("unexpected second broadcast result: %+v", result)
	}
	assertEvent(t, clientA.Send, EventTypeMessageNew)
}

func TestHubSubscribeUserAddsConversationToExistingConnections(t *testing.T) {
	hub := NewHub()
	clientA := &Client{UserID: "u1", Send: make(chan Event, 2)}
	clientB := &Client{UserID: "u1", Send: make(chan Event, 2)}
	clientC := &Client{UserID: "u2", Send: make(chan Event, 2)}

	if err := hub.Register(clientA, []string{"conv-a"}); err != nil {
		t.Fatalf("register clientA: %v", err)
	}
	if err := hub.Register(clientB, nil); err != nil {
		t.Fatalf("register clientB: %v", err)
	}
	if err := hub.Register(clientC, nil); err != nil {
		t.Fatalf("register clientC: %v", err)
	}

	if err := hub.SubscribeUser("u1", "conv-new"); err != nil {
		t.Fatalf("subscribe user: %v", err)
	}

	result, err := hub.BroadcastToConversation("conv-new", Event{Type: EventTypeMessageNew})
	if err != nil {
		t.Fatalf("broadcast conv-new: %v", err)
	}
	if result.Delivered != 2 || result.Dropped != 0 {
		t.Fatalf("unexpected broadcast result: %+v", result)
	}

	assertEvent(t, clientA.Send, EventTypeMessageNew)
	assertEvent(t, clientB.Send, EventTypeMessageNew)
	assertNoEvent(t, clientC.Send)
}

func TestHubValidationAndDropSemantics(t *testing.T) {
	hub := NewHub()

	if err := hub.Register(nil, nil); !errors.Is(err, ErrClientNil) {
		t.Fatalf("expected ErrClientNil, got %v", err)
	}

	clientNoSend := &Client{UserID: "u1"}
	if err := hub.Register(clientNoSend, nil); !errors.Is(err, ErrClientSendChannelNil) {
		t.Fatalf("expected ErrClientSendChannelNil, got %v", err)
	}

	clientNoUser := &Client{Send: make(chan Event, 1)}
	if err := hub.Register(clientNoUser, nil); !errors.Is(err, ErrClientUserIDRequired) {
		t.Fatalf("expected ErrClientUserIDRequired, got %v", err)
	}

	client := &Client{UserID: "u1", Send: make(chan Event)}
	if err := hub.Register(client, []string{"conv-a"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if _, err := hub.BroadcastToConversation("", Event{Type: EventTypeMessageDeleted}); !errors.Is(err, ErrConversationIDMissing) {
		t.Fatalf("expected ErrConversationIDMissing, got %v", err)
	}
	if _, err := hub.BroadcastToConversation("conv-a", Event{}); !errors.Is(err, ErrEventTypeMissing) {
		t.Fatalf("expected ErrEventTypeMissing, got %v", err)
	}

	result, err := hub.BroadcastToConversation("conv-a", Event{Type: EventTypeMessageDeleted})
	if err != nil {
		t.Fatalf("broadcast conv-a: %v", err)
	}
	if result.Delivered != 0 || result.Dropped != 1 {
		t.Fatalf("expected dropped event for blocked client, got %+v", result)
	}
}

func TestHubConversationIDs(t *testing.T) {
	hub := NewHub()
	client := &Client{UserID: "u1", Send: make(chan Event, 1)}
	if err := hub.Register(client, []string{"conv-b", "conv-a"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	conversationIDs, err := hub.ConversationIDs(client)
	if err != nil {
		t.Fatalf("conversation ids: %v", err)
	}
	if !reflect.DeepEqual(conversationIDs, []string{"conv-a", "conv-b"}) {
		t.Fatalf("unexpected conversation ids: %#v", conversationIDs)
	}
}

func assertEvent(t *testing.T, ch <-chan Event, eventType string) {
	t.Helper()
	select {
	case event := <-ch:
		if event.Type != eventType {
			t.Fatalf("expected event type %q, got %q", eventType, event.Type)
		}
	default:
		t.Fatalf("expected event %q, channel was empty", eventType)
	}
}

func assertNoEvent(t *testing.T, ch <-chan Event) {
	t.Helper()
	select {
	case event := <-ch:
		t.Fatalf("expected no event, got %+v", event)
	default:
	}
}

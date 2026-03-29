package ralphloop

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunTurnInterruptsWhenInactive(t *testing.T) {
	previous := turnInactivityTimeoutFn
	turnInactivityTimeoutFn = func(time.Duration) time.Duration { return 20 * time.Millisecond }
	defer func() {
		turnInactivityTimeoutFn = previous
	}()
	previousInterruptTimeout := turnInterruptTimeout
	turnInterruptTimeout = 20 * time.Millisecond
	defer func() {
		turnInterruptTimeout = previousInterruptTimeout
	}()

	interrupted := false
	client := &appServerClient{
		waitResult:    make(chan error),
		readErr:       make(chan error),
		notifications: make(chan jsonRPCNotification, 4),
		pending:       map[int64]chan jsonRPCEnvelope{},
		requestFn: func(_ context.Context, method string, _ map[string]any) (map[string]any, error) {
			switch method {
			case "turn/start":
				return map[string]any{"turn": map[string]any{"id": "turn-1"}}, nil
			case "turn/interrupt":
				interrupted = true
				return map[string]any{}, nil
			default:
				t.Fatalf("unexpected request method: %s", method)
				return nil, nil
			}
		},
	}

	_, err := client.RunTurn(context.Background(), runTurnOptions{
		ThreadID: "thread-1",
		Prompt:   "continue",
		Timeout:  time.Second,
	})
	if err == nil {
		t.Fatalf("expected inactivity error")
	}
	if !strings.Contains(err.Error(), "inactive") {
		t.Fatalf("expected inactivity error, got %v", err)
	}
	if !interrupted {
		t.Fatalf("expected turn interrupt request")
	}
}

func TestRunTurnReturnsWhenInterruptHangs(t *testing.T) {
	previous := turnInactivityTimeoutFn
	turnInactivityTimeoutFn = func(time.Duration) time.Duration { return 20 * time.Millisecond }
	defer func() {
		turnInactivityTimeoutFn = previous
	}()
	previousInterruptTimeout := turnInterruptTimeout
	turnInterruptTimeout = 20 * time.Millisecond
	defer func() {
		turnInterruptTimeout = previousInterruptTimeout
	}()

	interruptCalled := false
	client := &appServerClient{
		waitResult:    make(chan error),
		readErr:       make(chan error),
		notifications: make(chan jsonRPCNotification, 4),
		pending:       map[int64]chan jsonRPCEnvelope{},
		requestFn: func(ctx context.Context, method string, _ map[string]any) (map[string]any, error) {
			switch method {
			case "turn/start":
				return map[string]any{"turn": map[string]any{"id": "turn-1"}}, nil
			case "turn/interrupt":
				interruptCalled = true
				<-ctx.Done()
				return nil, ctx.Err()
			default:
				t.Fatalf("unexpected request method: %s", method)
				return nil, nil
			}
		},
	}

	start := time.Now()
	_, err := client.RunTurn(context.Background(), runTurnOptions{
		ThreadID: "thread-1",
		Prompt:   "continue",
		Timeout:  time.Second,
	})
	if err == nil {
		t.Fatalf("expected inactivity error")
	}
	if !strings.Contains(err.Error(), "inactive") {
		t.Fatalf("expected inactivity error, got %v", err)
	}
	if !interruptCalled {
		t.Fatalf("expected interrupt attempt")
	}
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("RunTurn took too long after hanging interrupt: %s", elapsed)
	}
}

func TestRunTurnResetsInactivityTimerOnNotification(t *testing.T) {
	previous := turnInactivityTimeoutFn
	turnInactivityTimeoutFn = func(time.Duration) time.Duration { return 20 * time.Millisecond }
	defer func() {
		turnInactivityTimeoutFn = previous
	}()

	client := &appServerClient{
		waitResult:    make(chan error),
		readErr:       make(chan error),
		notifications: make(chan jsonRPCNotification, 4),
		pending:       map[int64]chan jsonRPCEnvelope{},
		requestFn: func(_ context.Context, method string, _ map[string]any) (map[string]any, error) {
			switch method {
			case "turn/start":
				return map[string]any{"turn": map[string]any{"id": "turn-1"}}, nil
			case "turn/interrupt":
				t.Fatalf("did not expect interrupt")
				return nil, nil
			default:
				t.Fatalf("unexpected request method: %s", method)
				return nil, nil
			}
		},
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		client.notifications <- jsonRPCNotification{
			Method: "turn/started",
			Params: map[string]any{
				"turn": map[string]any{"id": "turn-1"},
			},
		}
		time.Sleep(10 * time.Millisecond)
		client.notifications <- jsonRPCNotification{
			Method: "turn/completed",
			Params: map[string]any{
				"turn": map[string]any{
					"id":     "turn-1",
					"status": "completed",
				},
			},
		}
	}()

	result, err := client.RunTurn(context.Background(), runTurnOptions{
		ThreadID: "thread-1",
		Prompt:   "continue",
		Timeout:  time.Second,
	})
	if err != nil {
		t.Fatalf("RunTurn returned error: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("status = %q", result.Status)
	}
	if result.TurnID != "turn-1" {
		t.Fatalf("turn id = %q", result.TurnID)
	}
}

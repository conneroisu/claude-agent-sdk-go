//go:build integration
// +build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
)

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Println("Skipping integration tests: claude CLI not found")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestQuery_BasicInteraction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	opts := &options.AgentOptions{
		MaxTurns: intPtr(1),
	}

	msgCh, errCh := claude.Query(ctx, "What is 2+2?", opts, nil)

	var gotResponse bool
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				if !gotResponse {
					t.Fatal("no response received")
				}
				return
			}

			if _, ok := msg.(*messages.AssistantMessage); ok {
				gotResponse = true
				t.Logf("Received: %+v", msg)
			}

		case err := <-errCh:
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			return

		case <-ctx.Done():
			t.Fatal("timeout")
		}
	}
}

func TestStreamingClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	client := claude.NewClient(&options.AgentOptions{
		MaxTurns: intPtr(1),
	})

	if err := client.Connect(ctx, nil); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	defer client.Close()

	if err := client.SendMessage(ctx, "Hello"); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	msgCh, errCh := client.ReceiveMessages(ctx)

	select {
	case msg := <-msgCh:
		t.Logf("Received: %+v", msg)

	case err := <-errCh:
		if err != nil {
			t.Fatalf("error: %v", err)
		}

	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func TestQuery_WithHooks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	var preToolUseCalled bool
	hooks := map[claude.HookEvent][]claude.HookMatcher{
		claude.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks: []claude.HookCallback{
					func(
						input map[string]any,
						toolUseID *string,
						ctx claude.HookContext,
					) (map[string]any, error) {
						preToolUseCalled = true
						return map[string]any{}, nil
					},
				},
			},
		},
	}

	opts := &options.AgentOptions{
		MaxTurns: intPtr(1),
	}

	msgCh, errCh := claude.Query(
		ctx,
		"List files in current directory",
		opts,
		hooks,
	)

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				if preToolUseCalled {
					t.Log("Hook was called successfully")
				}
				return
			}
			t.Logf("Received: %+v", msg)

		case err := <-errCh:
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			return

		case <-ctx.Done():
			t.Fatal("timeout")
		}
	}
}

func intPtr(i int) *int {
	return &i
}

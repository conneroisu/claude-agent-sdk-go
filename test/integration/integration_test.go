//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	claudeagent "github.com/conneroisu/claude-agent-sdk-go"
)

func TestBasicQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := claudeagent.NewClient(&claudeagent.Options{
		Model: "claude-sonnet-4-5",
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Send query
	err = client.Query(ctx, "What is 2+2? Just respond with the number.")
	if err != nil {
		t.Fatalf("Failed to send query: %v", err)
	}

	// Receive responses
	msgChan, errChan := client.ReceiveResponse(ctx)

	gotAssistantResponse := false
	gotResult := false

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				if !gotAssistantResponse {
					t.Error("Did not receive assistant response")
				}
				if !gotResult {
					t.Error("Did not receive result message")
				}
				return
			}

			switch m := msg.(type) {
			case *claudeagent.SDKAssistantMessage:
				gotAssistantResponse = true
				if len(m.Message.Content) == 0 {
					t.Error("Assistant response has no content")
				}
				t.Logf("Assistant responded with %d content blocks", len(m.Message.Content))

			case *claudeagent.SDKResultMessage:
				gotResult = true
				if m.Subtype != claudeagent.ResultSubtypeSuccess {
					t.Errorf("Expected success result, got %s", m.Subtype)
				}
				t.Logf("Query completed in %dms with cost $%.4f", m.DurationMS, m.TotalCostUSD)
			}

		case err := <-errChan:
			if err != nil {
				t.Fatalf("Error: %v", err)
			}

		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}
}

func TestAgentWithDisallowedTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a client with a custom agent that has disallowedTools
	client, err := claudeagent.NewClient(&claudeagent.Options{
		Model: "claude-sonnet-4-5",
		Agents: map[string]claudeagent.AgentDefinition{
			"restricted-agent": {
				Description:     "An agent that cannot use Bash or WebSearch",
				Prompt:          "You are a helpful assistant. You can read files but cannot execute bash commands or search the web.",
				DisallowedTools: []string{"Bash", "WebSearch"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Send query
	err = client.Query(ctx, "List the available tools you have access to.")
	if err != nil {
		t.Fatalf("Failed to send query: %v", err)
	}

	// Receive responses
	msgChan, errChan := client.ReceiveResponse(ctx)

	gotAssistantResponse := false
	gotResult := false

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				if !gotAssistantResponse {
					t.Error("Did not receive assistant response")
				}
				if !gotResult {
					t.Error("Did not receive result message")
				}
				return
			}

			switch m := msg.(type) {
			case *claudeagent.SDKAssistantMessage:
				gotAssistantResponse = true
				if len(m.Message.Content) == 0 {
					t.Error("Assistant response has no content")
				}
				t.Logf("Agent responded with %d content blocks", len(m.Message.Content))

			case *claudeagent.SDKResultMessage:
				gotResult = true
				if m.Subtype != claudeagent.ResultSubtypeSuccess {
					t.Errorf("Expected success result, got %s", m.Subtype)
				}
				t.Logf("Query completed in %dms with cost $%.4f", m.DurationMS, m.TotalCostUSD)
			}

		case err := <-errChan:
			if err != nil {
				t.Fatalf("Error: %v", err)
			}

		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}
}

func TestAgentWithToolsAllowlist(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a client with a custom agent that has a tools allowlist
	client, err := claudeagent.NewClient(&claudeagent.Options{
		Model: "claude-sonnet-4-5",
		Agents: map[string]claudeagent.AgentDefinition{
			"read-only-agent": {
				Description: "An agent that can only read files",
				Prompt:      "You are a read-only assistant. You can only read files, nothing else.",
				Tools:       []string{"Read", "Glob"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Send query
	err = client.Query(ctx, "What tools do you have?")
	if err != nil {
		t.Fatalf("Failed to send query: %v", err)
	}

	// Receive responses
	msgChan, errChan := client.ReceiveResponse(ctx)

	gotAssistantResponse := false
	gotResult := false

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				if !gotAssistantResponse {
					t.Error("Did not receive assistant response")
				}
				if !gotResult {
					t.Error("Did not receive result message")
				}
				return
			}

			switch m := msg.(type) {
			case *claudeagent.SDKAssistantMessage:
				gotAssistantResponse = true
				if len(m.Message.Content) == 0 {
					t.Error("Assistant response has no content")
				}
				t.Logf("Agent responded with %d content blocks", len(m.Message.Content))

			case *claudeagent.SDKResultMessage:
				gotResult = true
				if m.Subtype != claudeagent.ResultSubtypeSuccess {
					t.Errorf("Expected success result, got %s", m.Subtype)
				}
				t.Logf("Query completed in %dms with cost $%.4f", m.DurationMS, m.TotalCostUSD)
			}

		case err := <-errChan:
			if err != nil {
				t.Fatalf("Error: %v", err)
			}

		case <-ctx.Done():
			t.Fatal("Test timed out")
		}
	}
}

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

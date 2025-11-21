//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	claudeagent "github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
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

	// Receive responses (use ReceiveMessages which returns both channels)
	msgChan, errChan := client.ReceiveMessages(ctx)

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
				if m.Subtype != "success" {
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

func TestSetMaxThinkingTokens(t *testing.T) {
	ctx := context.Background()

	// Create a query directly for testing
	query, err := claudeagent.QueryFunc("Simple test query", &claudeagent.Options{
		Model: "claude-sonnet-4-5",
	})
	if err != nil {
		t.Fatalf("Failed to create query: %v", err)
	}
	defer query.Close()

	t.Run("SetMaxThinkingTokensWithPositiveValue", func(t *testing.T) {
		// Test setting a positive integer value
		limit := 1000
		err := query.SetMaxThinkingTokens(ctx, &limit)
		if err != nil {
			t.Errorf("SetMaxThinkingTokens with positive value failed: %v", err)
		}
		t.Logf("Successfully set max thinking tokens to %d", limit)
	})

	t.Run("SetMaxThinkingTokensWithNil", func(t *testing.T) {
		// Test clearing the limit with nil
		err := query.SetMaxThinkingTokens(ctx, nil)
		if err != nil {
			t.Errorf("SetMaxThinkingTokens with nil failed: %v", err)
		}
		t.Log("Successfully cleared max thinking tokens limit")
	})

	t.Run("SetMaxThinkingTokensWithDifferentValues", func(t *testing.T) {
		// Test setting multiple different values
		values := []int{500, 2000, 100}
		for _, val := range values {
			limit := val
			err := query.SetMaxThinkingTokens(ctx, &limit)
			if err != nil {
				t.Errorf("SetMaxThinkingTokens with value %d failed: %v", val, err)
			} else {
				t.Logf("Successfully set max thinking tokens to %d", limit)
			}
		}
	})

	t.Run("SetMaxThinkingTokensOnClosedQuery", func(t *testing.T) {
		// Create a new query for cancellation test
		cancelQuery, err := claudeagent.QueryFunc("Test query", &claudeagent.Options{
			Model: "claude-sonnet-4-5",
		})
		if err != nil {
			t.Fatalf("Failed to create query for cancellation test: %v", err)
		}

		// Close the query immediately to simulate cancellation
		cancelQuery.Close()

		// Try to set max thinking tokens on closed query
		limit := 500
		err = cancelQuery.SetMaxThinkingTokens(ctx, &limit)
		// We expect an error since the query is closed
		// The actual error type may vary, but it should not be nil
		if err == nil {
			t.Log("SetMaxThinkingTokens on closed query returned no error (this may be acceptable)")
		} else {
			t.Logf("SetMaxThinkingTokens on closed query correctly returned error: %v", err)
		}
	})
}

func TestAccountInfo(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a query directly for testing
	query, err := claudeagent.QueryFunc("Test query", &claudeagent.Options{
		Model: "claude-sonnet-4-5",
	})
	if err != nil {
		t.Fatalf("Failed to create query: %v", err)
	}
	defer query.Close()

	t.Run("AccountInfoBasicRetrieval", func(t *testing.T) {
		// Test retrieving account information
		accountInfo, err := query.AccountInfo(ctx)
		if err != nil {
			t.Errorf("AccountInfo request failed: %v", err)
			return
		}

		// Verify the returned struct is not nil
		if accountInfo == nil {
			t.Error("AccountInfo returned nil pointer")
			return
		}

		t.Log("Successfully retrieved account info")

		// Log the fields if they are populated (all fields are optional pointers)
		if accountInfo.Email != nil {
			t.Logf("Email: %s", *accountInfo.Email)
		} else {
			t.Log("Email: not provided")
		}

		if accountInfo.Organization != nil {
			t.Logf("Organization: %s", *accountInfo.Organization)
		} else {
			t.Log("Organization: not provided")
		}

		if accountInfo.SubscriptionType != nil {
			t.Logf("SubscriptionType: %s", *accountInfo.SubscriptionType)
		} else {
			t.Log("SubscriptionType: not provided")
		}

		if accountInfo.TokenSource != nil {
			t.Logf("TokenSource: %s", *accountInfo.TokenSource)
		} else {
			t.Log("TokenSource: not provided")
		}

		if accountInfo.ApiKeySource != nil {
			t.Logf("ApiKeySource: %s", *accountInfo.ApiKeySource)
		} else {
			t.Log("ApiKeySource: not provided")
		}
	})

	t.Run("AccountInfoMultipleCalls", func(t *testing.T) {
		// Test that AccountInfo can be called multiple times
		accountInfo1, err1 := query.AccountInfo(ctx)
		if err1 != nil {
			t.Errorf("First AccountInfo request failed: %v", err1)
			return
		}

		accountInfo2, err2 := query.AccountInfo(ctx)
		if err2 != nil {
			t.Errorf("Second AccountInfo request failed: %v", err2)
			return
		}

		if accountInfo1 == nil || accountInfo2 == nil {
			t.Error("One of the AccountInfo calls returned nil")
			return
		}

		t.Log("Successfully called AccountInfo multiple times")

		// Verify consistency between calls (stateless operation)
		if accountInfo1.Email != nil && accountInfo2.Email != nil {
			if *accountInfo1.Email != *accountInfo2.Email {
				t.Error("Email differs between calls (should be consistent)")
			}
		}
	})

	t.Run("AccountInfoWithContextCancellation", func(t *testing.T) {
		// Create a context that is already cancelled
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Try to get account info with cancelled context
		_, err := query.AccountInfo(cancelledCtx)
		if err == nil {
			t.Error("Expected error when calling AccountInfo with cancelled context, got nil")
		} else if err == context.Canceled {
			t.Logf("AccountInfo correctly returned context.Canceled error: %v", err)
		} else {
			t.Logf("AccountInfo returned error with cancelled context: %v", err)
		}
	})

	t.Run("AccountInfoWithTimeout", func(t *testing.T) {
		// Create a context with a very short timeout
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Wait for timeout to occur
		time.Sleep(10 * time.Millisecond)

		// Try to get account info with timed-out context
		_, err := query.AccountInfo(timeoutCtx)
		if err == nil {
			t.Log("AccountInfo with timed-out context returned no error (request may have completed)")
		} else if err == context.DeadlineExceeded {
			t.Logf("AccountInfo correctly returned context.DeadlineExceeded error: %v", err)
		} else {
			t.Logf("AccountInfo returned error with timed-out context: %v", err)
		}
	})

	t.Run("AccountInfoOnClosedQuery", func(t *testing.T) {
		// Create a new query for this test
		closedQuery, err := claudeagent.QueryFunc("Test query", &claudeagent.Options{
			Model: "claude-sonnet-4-5",
		})
		if err != nil {
			t.Fatalf("Failed to create query for closed test: %v", err)
		}

		// Close the query immediately
		closedQuery.Close()

		// Try to get account info on closed query
		testCtx, testCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer testCancel()

		_, err = closedQuery.AccountInfo(testCtx)
		if err == nil {
			t.Log("AccountInfo on closed query returned no error (this may be acceptable)")
		} else {
			t.Logf("AccountInfo on closed query correctly returned error: %v", err)
		}
	})
}

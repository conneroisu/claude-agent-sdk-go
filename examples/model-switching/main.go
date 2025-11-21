// Package main demonstrates model switching with Claude Agent SDK.
//
// Shows how to switch between models using SetModel() for cost and
// performance optimization. Start with fast models (Haiku) for simple
// queries, then switch to powerful models (Sonnet/Opus) for complex tasks.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
)

const (
	maxDemoTurns = 10
)

// main demonstrates cost and performance optimization via model switching.
// Starts with Haiku for simple tasks, switches to Sonnet for complex ones.
func main() {
	ctx := context.Background()
	fmt.Println("=== Model Switching Example ===")
	fmt.Println("Demonstrates switching between models for different tasks")
	fmt.Println()

	client, err := claude.NewClient(&claude.Options{
		Model:    "claude-haiku-4",
		MaxTurns: maxDemoTurns,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if e := client.Close(); e != nil {
			log.Printf("Failed to close client: %v", e)
		}
	}()

	h := runPhase1SimpleQuery(ctx, client)
	s := runPhase2ComplexQuery(ctx, client)
	runPhase3QueryModels(ctx, client)
	printSummary(h, s)
}

// modelStats holds performance statistics for a model.
type modelStats struct {
	cost     float64
	duration int
}

// runPhase1SimpleQuery demonstrates fast Haiku model for simple queries.
// Shows low-cost, high-speed responses for basic arithmetic.
func runPhase1SimpleQuery(ctx context.Context,
	client *claude.ClaudeSDKClient) modelStats {
	fmt.Println("--- Phase 1: Simple Query with Claude Haiku ---")
	fmt.Println("(Using fast model for quick responses)")
	err := client.Query(ctx, "What is 2 + 2? Just give me the number.")
	if err != nil {
		log.Printf("Failed to send query: %v", err)

		return modelStats{}
	}

	var stats modelStats
	for msg := range client.ReceiveResponse(ctx) {
		switch m := msg.(type) {
		case *claude.SDKSystemMessage:
			if m.Subtype == "init" {
				fmt.Printf("✓ Using model: %v\n", m.Data["model"])
			}
		case *claude.SDKAssistantMessage:
			fmt.Print("Claude Haiku: ")
			for _, block := range m.Message.Content {
				if tb, ok := block.(claude.TextBlock); ok {
					fmt.Print(tb.Text)
				} else if tcb, ok := block.(claude.TextContentBlock); ok {
					fmt.Print(tcb.Text)
				}
			}
			fmt.Println()
		case *claude.SDKResultMessage:
			stats.cost = m.TotalCostUSD
			stats.duration = m.DurationMS
		}
	}

	fmt.Printf("Haiku stats - Duration: %dms, Cost: $%.6f\n",
		stats.duration, stats.cost)
	fmt.Println()

	return stats
}

// runPhase2ComplexQuery switches to Sonnet for complex reasoning tasks.
// Demonstrates SetModel() for mid-conversation model switching.
func runPhase2ComplexQuery(ctx context.Context,
	client *claude.ClaudeSDKClient) modelStats {
	fmt.Println("--- Phase 2: Switching to Sonnet for Complex Task ---")
	model := "claude-sonnet-4-5" // More capable model for reasoning
	if err := client.SetModel(ctx, &model); err != nil {
		log.Printf("Failed to switch model: %v", err)

		return modelStats{}
	}
	fmt.Println("✓ Model switched to:", model)
	q := "Explain recursion with a simple example and code.\nBe brief."
	fmt.Println("Query:", q)
	fmt.Println()
	if err := client.Query(ctx, q); err != nil {
		log.Printf("Failed to send query: %v", err)

		return modelStats{}
	}

	var stats modelStats // Track cost and duration for comparison
	for msg := range client.ReceiveResponse(ctx) {
		switch m := msg.(type) {
		case *claude.SDKAssistantMessage:
			fmt.Println("Claude Sonnet:")
			for _, b := range m.Message.Content {
				if tb, ok := b.(claude.TextBlock); ok {
					fmt.Printf("  %s\n", tb.Text)
				} else if tcb, ok := b.(claude.TextContentBlock); ok {
					fmt.Printf("  %s\n", tcb.Text)
				}
			}
		case *claude.SDKResultMessage:
			stats.cost = m.TotalCostUSD
			stats.duration = m.DurationMS
		}
	}
	fmt.Printf("\nSonnet - Duration: %dms, Cost: $%.6f\n\n",
		stats.duration, stats.cost)

	return stats
}

// runPhase3QueryModels lists all available Claude models from the API.
func runPhase3QueryModels(ctx context.Context,
	client *claude.ClaudeSDKClient) {
	fmt.Println("--- Phase 3: Querying Available Models ---")
	models, err := client.SupportedModels(ctx)
	if err != nil {
		log.Printf("Warning: Could not fetch models: %v", err)

		return
	}
	fmt.Println("\nAvailable models:")
	for _, model := range models { // Display each model with description
		fmt.Printf("  - %s: %s\n", model.DisplayName, model.Description)
	}
}

// printSummary compares Haiku vs Sonnet performance and displays insights.
// Shows speed/cost trade-offs and optimization strategies.
func printSummary(h, s modelStats) {
	fmt.Println("\n=== Model Switching Summary ===")
	fmt.Printf("Haiku (simple):  %dms, $%.6f\n", h.duration, h.cost)
	fmt.Printf("Sonnet (complex): %dms, $%.6f\n", s.duration, s.cost)
	if s.duration > 0 && h.duration > 0 { // Calculate speed difference
		speedup := float64(s.duration) / float64(h.duration)
		fmt.Printf("\nHaiku was %.2fx faster\n", speedup)
	}
	if s.cost > 0 && h.cost > 0 { // Calculate cost difference
		fmt.Printf("Sonnet cost %.2fx more\n", s.cost/h.cost)
	}
	// Key insights from this example
	fmt.Println("\n=== Key Takeaways ===")
	fmt.Println("1. Use faster/cheaper models for simple queries")
	fmt.Println("2. Switch to powerful models for complex reasoning")
	fmt.Println("3. SetModel() allows dynamic optimization")
	fmt.Println("4. Cost/performance trade-offs managed in real-time")
	fmt.Println("\n✓ Intelligent cost and performance optimization!")
}

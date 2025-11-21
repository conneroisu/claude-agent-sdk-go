// Package main provides a file analyzer using the Claude Agent SDK.
//
// This tool demonstrates advanced SDK features:
//   - Using multiple tools (Read, Glob, Grep)
//   - Setting a custom working directory
//   - Handling long-running analysis tasks
//   - Tracking tool usage and displaying statistics
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/connerohnesorge/claude-agent-sdk-go/pkg/claude"
)

const (
	// maxAnalysisTurns is the maximum conversation turns for analysis.
	maxAnalysisTurns = 20
	// separatorWidth is the width of separator lines in output.
	separatorWidth = 60
	// minArgs is the minimum number of command line arguments required.
	minArgs = 2
)

func main() {
	if len(os.Args) < minArgs {
		printUsage()
		os.Exit(1)
	}

	absDir, err := validateAndResolveDir(os.Args[1])
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	ctx := context.Background()

	printHeader(absDir)

	client, err := createClient(absDir)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	defer closeClient(client)

	query := buildAnalysisQuery(absDir)
	startAnalysis()

	if err = client.Query(ctx, query); err != nil {
		log.Printf("Failed to send query: %v", err)

		return
	}

	processMessages(ctx, client)
}

// printUsage displays command usage information.
func printUsage() {
	fmt.Println("Usage: file-analyzer <directory>")
	fmt.Println("\nExample:\n  file-analyzer ./examples")
	fmt.Println("\nAnalyzes Go files in a directory using Claude.")
}

// validateAndResolveDir validates and resolves directory path.
func validateAndResolveDir(targetDir string) (string, error) {
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve directory: %w", err)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("directory error: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", absDir)
	}

	return absDir, nil
}

// printHeader displays the analysis header.
func printHeader(absDir string) {
	fmt.Println("Go File Analyzer\n================")
	fmt.Printf("Target directory: %s\n\n", absDir)
}

// createClient creates and configures the Claude Agent client.
func createClient(absDir string) (*claude.ClaudeSDKClient, error) {
	opts := &claude.Options{
		Model:    "claude-sonnet-4-5",
		MaxTurns: maxAnalysisTurns,
		Cwd:      absDir,
		AllowedTools: []string{
			"Read",
			"Glob",
			"Grep",
		},
	}

	return claude.NewClient(opts)
}

// closeClient safely closes the client connection.
func closeClient(client *claude.ClaudeSDKClient) {
	if closeErr := client.Close(); closeErr != nil {
		log.Printf("Failed to close client: %v", closeErr)
	}
}

// buildAnalysisQuery constructs the analysis query string.
func buildAnalysisQuery(absDir string) string {
	return fmt.Sprintf(
		`Analyze Go files in %s. Find all .go files (Glob "**/*.go"),
read each file, analyze purpose/exports/imports, and provide a
summary with file count, structure overview, and key components.`,
		absDir,
	)
}

// startAnalysis displays the analysis start message.
func startAnalysis() {
	sep := strings.Repeat("=", separatorWidth)
	fmt.Printf("Starting analysis...\n%s\n", sep)
}

// processMessages handles incoming messages from the client.
func processMessages(
	ctx context.Context,
	client *claude.ClaudeSDKClient,
) {
	msgChan, errChan := client.ReceiveMessages(ctx)
	tracker := &toolTracker{tools: make(map[string]bool)}

	for {
		select {
		case msg := <-msgChan:
			if msg == nil {
				return
			}

			processMessage(msg, tracker)

		case err := <-errChan:
			if err != nil {
				log.Printf("Error: %v", err)

				return
			}
		}
	}
}

type toolTracker struct {
	tools map[string]bool
}

func (t *toolTracker) add(name string) {
	if !t.tools[name] {
		t.tools[name] = true
	}
}

func (t *toolTracker) list() []string {
	result := make([]string, 0, len(t.tools))
	for tool := range t.tools {
		result = append(result, tool)
	}

	return result
}

// processMessage handles a single SDK message.
func processMessage(msg claude.SDKMessage, tracker *toolTracker) {
	switch m := msg.(type) {
	case *claude.SDKAssistantMessage:
		handleAssistant(m, tracker)
	case *claude.SDKResultMessage:
		handleResult(m, tracker)
	}
}

// handleAssistant processes assistant messages.
func handleAssistant(m *claude.SDKAssistantMessage, t *toolTracker) {
	for _, block := range m.Message.Content {
		switch b := block.(type) {
		case claude.TextBlock:
			if b.Text != "" {
				fmt.Println(b.Text)
			}
		case claude.TextContentBlock:
			if b.Text != "" {
				fmt.Println(b.Text)
			}
		case claude.ToolUseContentBlock:
			t.add(b.Name)
			fmt.Printf("\n[Using %s tool...]\n", b.Name)
		}
	}
}

// handleResult displays the final analysis results.
func handleResult(m *claude.SDKResultMessage, t *toolTracker) {
	fmt.Println(strings.Repeat("=", separatorWidth))
	fmt.Println("\n✓ Analysis complete")
	fmt.Printf("  Duration: %dms\n", m.DurationMS)
	fmt.Printf("  Cost: $%.4f\n", m.TotalCostUSD)
	fmt.Printf("  Turns: %d\n", m.NumTurns)

	if len(t.tools) > 0 {
		fmt.Printf("  Tools used: %s\n", strings.Join(t.list(), ", "))
	}

	if m.IsError {
		fmt.Printf("\n⚠ Analysis ended with error: %s\n", m.Subtype)
	}
}

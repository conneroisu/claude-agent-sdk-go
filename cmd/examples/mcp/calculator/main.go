// Package main demonstrates SDK MCP server with calculator tools.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/conneroisu/claude/pkg/claude"
	"github.com/conneroisu/claude/pkg/claude/messages"
	"github.com/conneroisu/claude/pkg/claude/options"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MathArgs defines arguments for math operations.
type MathArgs struct {
	A float64 `json:"a" jsonschema:"description=First number,required"`
	B float64 `json:"b" jsonschema:"description=Second number,required"`
}

// MathResult defines the result of math operations.
type MathResult struct {
	Result float64 `json:"result"`
}

func main() {
	ctx := context.Background()

	server := createCalculatorServer()

	opts := createAgentOptions(server)

	msgCh, errCh := claude.Query(
		ctx,
		"What is 15 + 27?",
		opts,
		nil,
	)

	processResponses(msgCh, errCh)
}

// createCalculatorServer creates and configures the calculator MCP server.
func createCalculatorServer() *mcpsdk.Server {
	server := claude.NewMCPServer("calculator", "1.0")

	claude.AddTool(server, &mcpsdk.Tool{
		Name:        "add",
		Description: "Add two numbers",
	}, addHandler)

	claude.AddTool(server, &mcpsdk.Tool{
		Name:        "subtract",
		Description: "Subtract two numbers",
	}, subtractHandler)

	claude.AddTool(server, &mcpsdk.Tool{
		Name:        "multiply",
		Description: "Multiply two numbers",
	}, multiplyHandler)

	claude.AddTool(server, &mcpsdk.Tool{
		Name:        "divide",
		Description: "Divide two numbers",
	}, divideHandler)

	return server
}

// createAgentOptions creates options with the calculator server.
func createAgentOptions(server *mcpsdk.Server) *options.AgentOptions {
	const defaultMaxTurns = 5
	maxTurns := defaultMaxTurns

	return &options.AgentOptions{
		MaxTurns: &maxTurns,
		MCPServers: map[string]options.MCPServerConfig{
			"calc": options.SDKServerConfig{
				Type:     "sdk",
				Name:     "calc",
				Instance: server,
			},
		},
		AllowedTools: []options.BuiltinTool{
			options.ToolMcp,
		},
	}
}

// addHandler implements the add tool.
func addHandler(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args MathArgs,
) (*mcpsdk.CallToolResult, MathResult, error) {
	return nil, MathResult{Result: args.A + args.B}, nil
}

// subtractHandler implements the subtract tool.
func subtractHandler(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args MathArgs,
) (*mcpsdk.CallToolResult, MathResult, error) {
	return nil, MathResult{Result: args.A - args.B}, nil
}

// multiplyHandler implements the multiply tool.
func multiplyHandler(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args MathArgs,
) (*mcpsdk.CallToolResult, MathResult, error) {
	return nil, MathResult{Result: args.A * args.B}, nil
}

// divideHandler implements the divide tool.
func divideHandler(
	_ context.Context,
	_ *mcpsdk.CallToolRequest,
	args MathArgs,
) (*mcpsdk.CallToolResult, MathResult, error) {
	if args.B == 0 {
		return &mcpsdk.CallToolResult{
			IsError: true,
		}, MathResult{}, nil
	}

	return nil, MathResult{Result: args.A / args.B}, nil
}

// processResponses handles messages and errors from the query.
func processResponses(
	msgCh <-chan messages.Message,
	errCh <-chan error,
) {
	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return
			}
			displayMessage(msg)

		case err, ok := <-errCh:
			if !ok {
				return
			}
			if err != nil {
				handleError(err)
			}
		}
	}
}

// handleError logs and exits on error.
//
//nolint:revive // Example code - deep-exit acceptable
func handleError(err error) {
	log.Fatalf("Error: %v", err)
}

// displayMessage prints a message to the console.
func displayMessage(msg messages.Message) {
	switch m := msg.(type) {
	case *messages.AssistantMessage:
		for _, block := range m.Content {
			printBlock(block)
		}

	case *messages.ResultMessageSuccess:
		printResult(m)
	}
}

// printBlock prints a content block.
func printBlock(block messages.ContentBlock) {
	if text, ok := block.(messages.TextBlock); ok {
		fmt.Printf("Claude: %s\n", text.Text)
	}
}

// printResult prints the final result.
func printResult(m *messages.ResultMessageSuccess) {
	fmt.Printf("\nResult: %s\n", m.Result)
	fmt.Printf("Cost: $%.6f\n", m.TotalCostUSD)
	fmt.Printf("Turns: %d\n", m.NumTurns)
}

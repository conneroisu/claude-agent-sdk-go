// Package claude provides tests for the Claude client and its components.
// This file contains tests for MCP server initialization logic.
package claude

import (
	"context"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/options"
)

// mockMCPServerInstance is a mock MCP server instance for testing.
// It implements the MessageHandler interface required by the MCP adapter.
type mockMCPServerInstance struct{}

// HandleMessage implements the MessageHandler interface for testing.
// The receiver is intentionally unused as this is a simple echo mock.
func (*mockMCPServerInstance) HandleMessage(
	_ context.Context,
	message []byte,
) ([]byte, error) {
	return message, nil
}

// TestInitializeMCPServers_EmptyAndNonSDK tests initialization behavior
// with empty configs and non-SDK server types.
func TestInitializeMCPServers_EmptyAndNonSDK(t *testing.T) {
	t.Run("returns nil for empty config", func(t *testing.T) {
		// Initialize with nil config
		servers, err := initializeMCPServers(nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// Verify nil is returned for empty config
		if servers != nil {
			t.Errorf("expected nil servers, got %v", servers)
		}
	})

	t.Run("skips non-SDK servers", func(t *testing.T) {
		// Create configs with only stdio and SSE servers
		configs := map[string]options.MCPServerConfig{
			"stdio-server": options.StdioServerConfig{
				Type:    "stdio",
				Command: "node",
				Args:    []string{"server.js"},
			},
			"sse-server": options.SSEServerConfig{
				Type: "sse",
				URL:  "http://localhost:3000",
			},
		}

		// Initialize and verify non-SDK servers are skipped
		servers, err := initializeMCPServers(configs)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// No SDK servers means nil result
		if servers != nil {
			t.Errorf(
				"expected nil servers (no SDK servers), got %v",
				servers,
			)
		}
	})
}

// TestInitializeMCPServers_SDKServers tests initialization of SDK-managed
// MCP servers with valid and invalid configurations.
func TestInitializeMCPServers_SDKServers(t *testing.T) {
	t.Run("initializes SDK server", func(t *testing.T) {
		// Create mock instance
		mockInstance := &mockMCPServerInstance{}
		configs := map[string]options.MCPServerConfig{
			"my-server": options.SDKServerConfig{
				Type:     "sdk",
				Name:     "my-server",
				Instance: mockInstance,
			},
		}

		// Initialize servers
		servers, err := initializeMCPServers(configs)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify server map is created
		if servers == nil {
			t.Fatal("expected servers map, got nil")
		}

		// Verify server exists in map
		server, exists := servers["my-server"]
		if !exists {
			t.Fatal("expected 'my-server' in servers map")
		}

		// Verify server name matches
		if server.Name() != "my-server" {
			t.Errorf(
				"expected server name 'my-server', got '%s'",
				server.Name(),
			)
		}
	})

	t.Run("returns error for SDK server with nil instance", func(t *testing.T) {
		// Create config with nil instance
		configs := map[string]options.MCPServerConfig{
			"bad-server": options.SDKServerConfig{
				Type:     "sdk",
				Name:     "bad-server",
				Instance: nil,
			},
		}

		// Verify error is returned for nil instance
		_, err := initializeMCPServers(configs)
		if err == nil {
			t.Fatal("expected error for nil instance, got nil")
		}
	})
}

// TestInitializeMCPServers_Multiple tests initialization with multiple
// servers of different types to verify filtering and initialization.
func TestInitializeMCPServers_Multiple(t *testing.T) {
	t.Run("initializes multiple SDK servers", func(t *testing.T) {
		// Create two mock instances
		mock1 := &mockMCPServerInstance{}
		mock2 := &mockMCPServerInstance{}

		// Create mixed config with SDK and non-SDK servers
		configs := map[string]options.MCPServerConfig{
			"server1": options.SDKServerConfig{
				Type:     "sdk",
				Name:     "server1",
				Instance: mock1,
			},
			"server2": options.SDKServerConfig{
				Type:     "sdk",
				Name:     "server2",
				Instance: mock2,
			},
			"stdio-server": options.StdioServerConfig{
				Type:    "stdio",
				Command: "node",
			},
		}

		// Initialize all servers
		servers, err := initializeMCPServers(configs)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify only SDK servers are initialized
		if len(servers) != 2 {
			t.Errorf("expected 2 servers, got %d", len(servers))
		}

		// Verify both SDK servers exist
		if _, exists := servers["server1"]; !exists {
			t.Error("expected 'server1' in servers map")
		}

		if _, exists := servers["server2"]; !exists {
			t.Error("expected 'server2' in servers map")
		}

		// Verify stdio server was skipped
		if _, exists := servers["stdio-server"]; exists {
			t.Error("stdio-server should not be in servers map")
		}
	})
}

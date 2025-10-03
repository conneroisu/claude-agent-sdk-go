package claude

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/conneroisu/claude/pkg/claude/adapters/mcp"
	"github.com/conneroisu/claude/pkg/claude/options"
	"github.com/conneroisu/claude/pkg/claude/ports"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// initializeMCPServers creates MCP client connections from configuration
// Returns a map of server name -> connected MCP client adapter.
func initializeMCPServers(
	ctx context.Context,
	configs map[string]options.MCPServerConfig,
) (map[string]ports.MCPServer, error) {
	if len(configs) == 0 {
		return nil, nil
	}

	servers := make(map[string]ports.MCPServer, len(configs))

	for name, cfg := range configs {
		server, err := initializeMCPServer(ctx, name, cfg)
		if err != nil {
			// Clean up already-connected servers
			for _, s := range servers {
				_ = s.Close()
			}

			return nil, fmt.Errorf("failed to initialize MCP server %q: %w", name, err)
		}
		servers[name] = server
	}

	return servers, nil
}

// initializeMCPServer creates a single MCP client connection.
func initializeMCPServer(
	ctx context.Context,
	name string,
	cfg options.MCPServerConfig,
) (ports.MCPServer, error) {
	var transport mcpsdk.Transport

	switch config := cfg.(type) {
	case options.StdioServerConfig:
		// Create stdio transport using command
		cmd := exec.CommandContext(ctx, config.Command, config.Args...)
		if config.Env != nil {
			cmd.Env = append(cmd.Env, mapToEnvSlice(config.Env)...)
		}
		transport = &mcpsdk.CommandTransport{Command: cmd}

	case options.HTTPServerConfig:
		// Create HTTP streamable transport
		// Note: Headers would need to be set via custom HTTPClient if required
		transport = &mcpsdk.StreamableClientTransport{
			Endpoint: config.URL,
		}

	case options.SSEServerConfig:
		// SSE uses same streamable transport as HTTP
		// Note: Headers would need to be set via custom HTTPClient if required
		transport = &mcpsdk.StreamableClientTransport{
			Endpoint: config.URL,
		}

	case options.SDKServerConfig:
		// SDK-managed servers are handled by the MCP adapter layer
		// This case should be handled by a separate registry/factory
		return nil, fmt.Errorf("SDK-managed MCP servers not yet implemented")

	default:
		return nil, fmt.Errorf("unknown MCP server config type: %T", cfg)
	}

	// Create MCP client using official SDK
	client := mcpsdk.NewClient(
		&mcpsdk.Implementation{
			Name:    "claude-agent-sdk-go",
			Version: "0.1.0",
		},
		nil,
	)

	// Connect to the MCP server
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	// Wrap the session in our adapter that implements ports.MCPServer
	return mcp.NewAdapter(name, session), nil
}

// mapToEnvSlice converts map[string]string to []string in KEY=VALUE format.
func mapToEnvSlice(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}

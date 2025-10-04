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

// initializeMCPServers creates MCP client connections from configuration.
// It returns a map of server name to connected MCP client adapter.
// All servers are initialized in sequence, and if any fail, all
// previously connected servers are cleaned up before returning error.
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
			cleanupMCPServers(servers)
			return nil, fmt.Errorf("failed to initialize MCP server %q: %w", name, err)
		}
		servers[name] = server
	}

	return servers, nil
}

// initializeMCPServer creates a single MCP client connection.
// It handles all four config types: Stdio, HTTP, SSE, and SDK.
func initializeMCPServer(
	ctx context.Context,
	name string,
	cfg options.MCPServerConfig,
) (ports.MCPServer, error) {
	switch config := cfg.(type) {
	case options.StdioServerConfig:
		return initStdioServer(ctx, name, config)
	case options.HTTPServerConfig:
		return initHTTPServer(ctx, name, config)
	case options.SSEServerConfig:
		return initSSEServer(ctx, name, config)
	case options.SDKServerConfig:
		return initSDKServer(ctx, name, config)
	default:
		return nil, fmt.Errorf("unknown MCP server config type: %T", cfg)
	}
}

// initStdioServer creates an MCP server connection via stdio transport.
func initStdioServer(
	ctx context.Context,
	name string,
	config options.StdioServerConfig,
) (ports.MCPServer, error) {
	cmd := exec.CommandContext(ctx, config.Command, config.Args...)
	if config.Env != nil {
		cmd.Env = append(cmd.Env, mapToEnvSlice(config.Env)...)
	}

	transport := &mcpsdk.CommandTransport{Command: cmd}
	return connectMCPClient(ctx, name, transport)
}

// initHTTPServer creates an MCP server connection via HTTP transport.
func initHTTPServer(
	ctx context.Context,
	name string,
	config options.HTTPServerConfig,
) (ports.MCPServer, error) {
	transport := &mcpsdk.StreamableClientTransport{
		Endpoint: config.URL,
	}
	// TODO: Add support for config.Headers via custom HTTPClient
	return connectMCPClient(ctx, name, transport)
}

// initSSEServer creates an MCP server connection via SSE transport.
func initSSEServer(
	ctx context.Context,
	name string,
	config options.SSEServerConfig,
) (ports.MCPServer, error) {
	transport := &mcpsdk.SSEClientTransport{
		Endpoint: config.URL,
	}
	// TODO: Add support for config.Headers via custom HTTPClient
	return connectMCPClient(ctx, name, transport)
}

// initSDKServer wraps a user-provided SDK server instance.
func initSDKServer(
	_ context.Context,
	name string,
	config options.SDKServerConfig,
) (ports.MCPServer, error) {
	return mcp.NewServerAdapter(name, config.Instance), nil
}

// connectMCPClient establishes a connection to an MCP server.
func connectMCPClient(
	ctx context.Context,
	name string,
	transport mcpsdk.Transport,
) (ports.MCPServer, error) {
	client := mcpsdk.NewClient(&mcpsdk.Implementation{
		Name:    "claude-agent-sdk-go",
		Version: "0.1.0",
	}, nil)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return mcp.NewClientAdapter(name, session), nil
}

// cleanupMCPServers closes all MCP server connections.
// Used for cleanup when initialization fails partway through.
func cleanupMCPServers(servers map[string]ports.MCPServer) {
	for _, server := range servers {
		_ = server.Close()
	}
}

// mapToEnvSlice converts a map to environment variable format.
// Returns a slice of "KEY=VALUE" strings suitable for exec.Cmd.Env.
func mapToEnvSlice(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

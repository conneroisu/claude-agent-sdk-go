## Hexagonal Architecture Summary

### Dependency Flow

The SDK strictly follows the dependency rule of hexagonal architecture:

```
LAYER 4: Public API (client.go, query.go)     
- Entry point for SDK users                    
- Wires domain services with adapters          
depends on ↓
LAYER 3: ADAPTERS (adapters/*)                 
- cli/      → implements ports.Transport       
- jsonrpc/  → implements ports.ProtocolHandler 
- parse/    → implements ports.MessageParser   
- mcp/      → implements ports.MCPServer       
depends on ↓
LAYER 2: PORTS (ports/*)                       
- Interfaces defined BY domain needs           
- Contract layer between domain and infra      
depends on ↓
LAYER 1: CORE DOMAIN (querying/, streaming/)  
- Pure business logic                          
- No infrastructure dependencies               
- Uses port interfaces, never adapters         
```

### Key Architectural Decisions

#### 1. Ports Define Contracts

All interfaces are defined in `ports/` based on domain needs:

- `ports.Transport` - What domain needs for I/O
- `ports.ProtocolHandler` - What domain needs for control protocol
- `ports.MessageParser` - What domain needs for parsing
- `ports.MCPServer` - What domain needs from MCP servers

#### 2. Adapters Implement Ports

Infrastructure code implements these interfaces:

- `adapters/cli.Adapter` implements `ports.Transport`
- `adapters/jsonrpc.Adapter` implements `ports.ProtocolHandler`
- `adapters/parse.Adapter` implements `ports.MessageParser`
- `adapters/mcp.Adapter` implements `ports.MCPServer`

#### 3. Domain Services Are Pure

Domain services contain ONLY business logic:

- `querying.Service` - Executes one-shot queries
- `streaming.Service` - Manages bidirectional conversations
- `hooking.Service` - Executes lifecycle hooks
- `permissions.Service` - Checks tool permissions
  NO protocol state management (request IDs, timeouts, etc.) in domain.

#### 4. Infrastructure Concerns Stay in Adapters

Control protocol state management is in `adapters/jsonrpc`:

- Pending request tracking
- Request ID generation
- Timeout handling
- Response routing

#### 5. Configuration Separation

Options are split by concern:

- `options/domain.go` - Pure domain config (PermissionMode, AgentDefinition)
- `options/transport.go` - Infrastructure config (Cwd, Env, MaxBufferSize)
- `options/mcp.go` - MCP server configurations

### Benefits of This Architecture

1. Testability

- Domain services testable without infrastructure
- Mock adapters via interfaces
- No subprocess spawning in unit tests

2. Flexibility

- Swap CLI transport for HTTP transport
- Change JSON-RPC to gRPC
- Add new message parsers

3. Clarity

- Clear boundaries between layers
- Easy to understand where code belongs
- Package names describe purpose

4. Maintainability

- Infrastructure changes don't affect domain
- Domain changes don't ripple to all adapters
- Each layer has single responsibility

### Compile-Time Guarantees

All adapters verify interface compliance at compile time:

```go
// adapters/cli/transport.go
var _ ports.Transport = (*Adapter)(nil)
// adapters/jsonrpc/protocol.go
var _ ports.ProtocolHandler = (*Adapter)(nil)
// adapters/parse/parser.go
var _ ports.MessageParser = (*Adapter)(nil)
```

If an adapter doesn't fully implement its port interface, the code won't compile.

## Architecture Overview

### Core Design Principles

- Idiomatic Go: Use Go conventions (interfaces, channels, contexts, errors)
- Type Safety: Leverage Go's strong typing with generics where appropriate
- Concurrency: Use goroutines and channels for async operations
- Error Handling: Explicit error returns following Go best practices
- Context Support: Full context.Context integration for cancellation and timeouts
- Zero Dependencies: Minimize external dependencies where possible
- Hexagonal Architecture: Strict separation between domain logic and infrastructure

### Hexagonal Architecture (Ports and Adapters)

This SDK follows hexagonal architecture principles, also known as ports and adapters pattern.

This architectural style isolates the core business logic (domain) from external concerns (infrastructure) by defining clear boundaries and dependency rules.

Four Key Principles:

- Domain Independence: Core domain never imports adapters or infrastructure code
- Ports Define Contracts: Interfaces defined by domain needs, not external systems
- Adapters Implement Ports: Infrastructure code implements domain-defined interfaces
- Dependency Direction: Always flows inward (adapters → domain), never outward

### Package Structure (Hexagonal Architecture)

Following hexagonal architecture principles (ports and adapters), the SDK separates the core domain from external dependencies. Package names describe what they provide (functionality/context), not what they contain (generic types).

```
claude-agent-sdk-go/
├── cmd/                        # ═══ BINARIES (Entry Points) ═══
│   └── examples/               # Example applications
│       ├── quickstart/
│       ├── streaming/
│       ├── hooks/
│       └── mcp/
│
├── pkg/claude/
│   # LAYER 1: CORE DOMAIN (Business Logic)
│   # - Never imports from adapters/
│   # - Only imports from ports/ (interfaces it defines)
│   # - Pure business logic, no infrastructure concerns
│   ├── querying/               # Domain service: "Execute one-shot queries"
│   │   └── service.go
│   ├── streaming/              # Domain service: "Manage streaming conversations"
│   │   └── service.go
│   ├── hooking/                # Domain service: "Execute lifecycle hooks"
│   │   └── service.go
│   ├── permissions/            # Domain service: "Check tool permissions"
│   │   └── service.go
│   │
│   # LAYER 1B: DOMAIN MODELS
│   # - Shared types used across domain
│   # - No infrastructure dependencies
│   ├── messages/               # Domain models: Message types
│   │   └── messages.go
│   ├── options/                # Domain models: Configuration
│   │   ├── domain.go           # Pure domain options (PermissionMode, etc.)
│   │   ├── transport.go        # Transport configuration
│   │   └── mcp.go              # MCP server configuration
│   │
│   # LAYER 2: PORTS (Domain-Defined Interfaces)
│   # - Interfaces defined BY domain needs
│   # - NOT defined by external systems
│   # - This is the "contract" layer
│   ├── ports/
│   │   ├── transport.go        # What domain needs from transport
│   │   ├── protocol.go         # What domain needs from control protocol
│   │   ├── parser.go           # What domain needs from message parsing
│   │   └── mcp.go              # What domain needs from MCP servers
│   │
│   # LAYER 3: ADAPTERS (Infrastructure Implementations)
│   # - Implements port interfaces
│   # - Handles external concerns (CLI, JSON-RPC, parsing)
│   # - Can import from domain and ports
│   # - Domain NEVER imports from here
│   ├── adapters/
│   │   ├── cli/                # Adapter: CLI subprocess transport
│   │   │   └── transport.go    # Implements ports.Transport
│   │   ├── jsonrpc/            # Adapter: Control protocol handler
│   │   │   └── protocol.go     # Implements ports.ProtocolHandler
│   │   ├── parse/              # Adapter: Message parser
│   │   │   └── parser.go       # Implements ports.MessageParser
│   │   └── mcp/                # Adapter: MCP server implementation
│   │       └── server.go       # Implements ports.MCPServer
│   │
│   # LAYER 4: PUBLIC API (Facade)
│   # - Wires domain services with adapters
│   # - Entry point for SDK users
│   # - Hides complexity of ports/adapters from users
│   ├── client.go               # Client for interactive conversations
│   ├── query.go                # Query() for one-shot requests
│   └── errors.go               # Public error types
│
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
├── LICENSE
└── .golangci.yaml
```

Hexagonal Architecture Key Principles:

1. Core Domain Independence: Domain packages (`querying`, `streaming`, `hooking`, `permissions`) contain business logic and do NOT import adapters
1. Ports Define Contracts: The `ports/` package contains interfaces defined by the domain's needs
1. Adapters Implement Ports: Adapters in `adapters/` implement the port interfaces and handle external concerns
1. Dependency Direction: Always flows inward - adapters depend on domain, never the reverse
1. Context-Based Naming: Packages named for what they provide (`querying`, `streaming`) not what they contain (`models`, `handlers`)

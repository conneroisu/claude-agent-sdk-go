## Key Design Decisions
### Hexagonal Architecture Principles
1. Domain Independence: Core domain packages (`querying`, `streaming`, `hooking`, `permissions`) never import adapters
2. Ports Define Contracts: Interfaces in `ports/` package are defined by domain needs, not external systems
3. Adapters Implement Ports: All infrastructure code in `adapters/` implements port interfaces
4. Dependency Direction: Always flows inward (adapters → domain), never outward (domain → adapters)
5. Package Naming: Named for what they provide (`querying`, `streaming`) not what they contain (`models`, `handlers`)
### Go Idioms
6. Channels vs Iterators: Use channels for async message streaming (idiomatic Go)
7. Context Integration: Full context.Context support throughout
8. Error Handling: Return errors explicitly, use error wrapping
9. Interface Compliance: Use `var _ ports.Transport = (*Adapter)(nil)` pattern to verify at compile time
10. Async Model: Goroutines + channels (Go's native async)
11. JSON Handling: Use encoding/json with struct tags
12. Testing Strategy: Table-driven tests, interface mocks, integration tests
### Architectural Benefits
- Testability: Domain logic testable without infrastructure dependencies
- Flexibility: Easy to swap adapters (e.g., different transport mechanisms)
- Clarity: Clear separation between business logic and technical details
- Maintainability: Changes to infrastructure don't affect domain
- Discoverability: Package names describe purpose at a glance
## Phase 3: Adapters (Infrastructure)

**Purpose:** Infrastructure adapters connect domain services to external systems. These components handle all operational failure modes, resource management, and production diagnostics.

**Priority:** Critical - These adapters contain the highest operational risk since they interface with external processes, I/O streams, and protocol state.

---

### 3.1 CLI Transport Adapter

**Package:** `adapters/cli/`

**Implements:** `ports.Transport`

**Responsibility:** Manages Claude CLI subprocess lifecycle, I/O streaming, and process health monitoring.

#### 3.1.1 CLI Discovery - Failure Modes & Mitigation

**What Can Go Wrong:**

1. **CLI not in PATH** - User installed via npm/yarn but binary not accessible
2. **Multiple CLI versions** - Different versions in PATH vs local node_modules
3. **Windows executable extensions** - Missing `.cmd`, `.bat`, `.exe` handling
4. **Permission denied** - CLI exists but not executable (common on Unix after npm install)
5. **Symlink resolution** - CLI is symlink pointing to invalid target
6. **Non-standard installations** - Custom npm prefix, global installations in unusual locations

**Detection Strategy:**

```go
// Discovery phase returns detailed diagnostic information
type CLIDiscoveryResult struct {
    Path           string
    Version        string        // Extracted from --version
    Executable     bool          // Actual execution test passed
    DiagnosticInfo string        // Human-readable context for errors
}

// Test execution with timeout to detect hanging/broken installs
func verifyExecutable(path string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, path, "--version")
    output, err := cmd.CombinedOutput()
    // Log version mismatch warnings if SDK expects specific CLI features
    return err
}
```

**Platform-Specific Fallbacks:**

- **Windows:** Check `.exe`, `.cmd`, `.bat` extensions; handle both forward/backslash paths
- **macOS:** Check `/usr/local/bin` (Homebrew), `~/.npm-global/bin`
- **Linux:** Check `~/.local/bin`, `/opt/`, distro-specific package paths
- **All platforms:** Support `CLAUDE_CLI_PATH` environment variable override for CI/testing

**Observability Hooks:**

- Log all discovery attempts (PATH search, fallback locations)
- Emit warning if multiple CLI binaries found (version confusion risk)
- Include full discovery trace in connection errors for user debugging

#### 3.1.2 Process Lifecycle - Failure Modes & Recovery

**What Can Go Wrong:**

1. **Process exits before ready** - CLI crashes immediately (bad config, missing dependencies)
2. **Zombie processes** - Subprocess orphaned due to parent crash/signal
3. **Resource leaks** - Pipes not closed, goroutines not stopped on error paths
4. **Signal handling** - Parent receives SIGTERM but child continues running
5. **Graceful shutdown timeout** - CLI doesn't respond to stdin close within reasonable time

**Process Health Monitoring:**

```go
// Track process state for diagnostics
type ProcessHealth struct {
    PID            int
    StartTime      time.Time
    LastActivity   time.Time  // Updated on each message read/write
    BytesRead      int64
    BytesWritten   int64
    StderrLines    int        // Count for saturation detection
}

// Detect stuck/hung processes
func (a *Adapter) monitorHealth(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if time.Since(a.health.LastActivity) > 2*time.Minute {
                // Emit warning: process appears hung
                a.emitHealthWarning("no activity for 2 minutes")
            }
        }
    }
}
```

**Graceful Shutdown Strategy:**

1. Close stdin to signal CLI to finish current turn
2. Wait up to 10s for process to exit naturally
3. Send SIGTERM if still running after 10s
4. Send SIGKILL if still running after 15s (last resort)
5. Close all pipes and channels in defer blocks to prevent goroutine leaks

**Zombie Prevention:**

- Use `exec.CommandContext` with root context to ensure process termination on parent exit
- Register signal handlers to propagate SIGTERM/SIGINT to child
- Track subprocess PID and verify termination in Close() method

#### 3.1.3 I/O Streaming - Buffer Saturation & Backpressure

**What Can Go Wrong:**

1. **Stdout buffer saturation** - Claude sends massive response exceeding buffer
2. **Stderr flood** - CLI emits excessive debug logs, blocking stderr reader
3. **Stdin write blocking** - Writing to stdin blocks forever if process hung
4. **Partial JSON lines** - Large messages split across multiple scanner reads
5. **Invalid UTF-8** - Corrupted output causes scanner/JSON parser errors
6. **Race between readers** - Stdout/stderr goroutines compete for resources

**Buffer Management:**

```go
// Configurable buffer limits with overflow detection
const (
    DefaultMaxBufferSize = 1 * 1024 * 1024      // 1MB
    MaxStderrBufferSize  = 256 * 1024           // 256KB
    DefaultChannelBuffer = 10                   // Buffered channels
)

// Implement backpressure for large responses
func (a *Adapter) ReadMessages(ctx context.Context) (<-chan map[string]any, <-chan error) {
    // Channel sizing prevents unbounded memory growth
    msgCh := make(chan map[string]any, DefaultChannelBuffer)

    // Scanner with size limit prevents single-message OOM
    scanner.Buffer(initialBuf, a.maxBufferSize)

    // Incremental buffer growth with limit
    if len(buffer) > a.maxBufferSize {
        return fmt.Errorf("message exceeded %d bytes - possible CLI corruption", a.maxBufferSize)
    }
}
```

**Stderr Handling:**

- **Non-blocking consumption:** Stderr reader must never block stdout processing
- **Rate limiting:** If stderr exceeds 100 lines/second, enable sampling (log every 10th line)
- **Overflow detection:** Track stderr volume; warn if exceeds 10MB total
- **Callback timeout:** If user stderr callback blocks >1s, skip further invocations and warn

**Partial Message Handling:**

```go
// Accumulate multi-line JSON safely
buffer := ""
for scanner.Scan() {
    buffer += scanner.Text()

    // Attempt parse after each line
    var msg map[string]any
    if err := json.Unmarshal([]byte(buffer), &msg); err == nil {
        buffer = "" // Reset on successful parse
        msgCh <- msg
    } else if len(buffer) > maxBufferSize {
        return fmt.Errorf("incomplete message exceeded buffer limit")
    }
    // Continue accumulating if JSON incomplete
}
```

#### 3.1.4 Error Propagation & Production Diagnostics

**Timeout Strategy:**

- **Connection timeout:** 30s for initial process startup + CLI handshake
- **Write timeout:** 5s per stdin write (detect broken pipe quickly)
- **Read timeout:** None on ReadMessages (streaming is indefinite), but monitor health
- **Shutdown timeout:** 10s for graceful close, 15s before SIGKILL

**Error Context Enrichment:**

```go
// Attach full diagnostic context to all errors
type CLIError struct {
    Stage       string                  // "discovery", "connect", "write", "read"
    Cause       error                   // Underlying error
    ProcessInfo *ProcessHealth          // PID, runtime stats
    CLIPath     string
    Command     []string                // Full command-line for reproduction
    Environment map[string]string       // Relevant env vars
}

func (e *CLIError) Error() string {
    return fmt.Sprintf("CLI %s failed: %v [pid=%d, runtime=%s, path=%s]",
        e.Stage, e.Cause, e.ProcessInfo.PID,
        time.Since(e.ProcessInfo.StartTime), e.CLIPath)
}
```

**Observability Hooks (Logging/Metrics):**

- **Startup:** Log CLI path, version, full command, process PID
- **I/O activity:** Increment counters for messages sent/received, bytes transferred
- **Health checks:** Emit gauge for time-since-last-activity
- **Errors:** Log full CLIError with structured fields for aggregation
- **Shutdown:** Log process exit code, runtime duration, total message count

**Suggested Instrumentation Points:**

```go
type TransportMetrics interface {
    RecordCLIDiscovery(path string, durationMs int64, success bool)
    RecordProcessStart(pid int)
    RecordMessageSent(bytes int)
    RecordMessageReceived(bytes int)
    RecordError(stage string, err error)
    RecordProcessExit(exitCode int, runtimeMs int64)
}

// Adapter optionally accepts metrics interface
func NewAdapter(opts *options.AgentOptions, metrics TransportMetrics) *Adapter
```

#### 3.1.5 Command Construction - Edge Cases

**Illustrative Example (not exhaustive):**

```go
// BuildCommand constructs CLI arguments, handling edge cases
func (a *Adapter) BuildCommand() ([]string, error) {
    cmd := []string{a.cliPath, "--output-format", "stream-json"}

    // Handle spaces in arguments (e.g., system prompts with quotes)
    if a.options.SystemPrompt != nil {
        // Shell escaping handled by exec.Command, but log for transparency
        cmd = append(cmd, "--system-prompt", escapeForLogging(systemPrompt))
    }

    // Validate file paths before passing to CLI
    if a.options.Cwd != nil {
        if !filepath.IsAbs(*a.options.Cwd) {
            return nil, fmt.Errorf("Cwd must be absolute path, got: %s", *a.options.Cwd)
        }
    }

    return cmd, nil
}
```

**Windows Path Handling:**

- Convert backslashes to forward slashes for `--add-dir` if CLI expects Unix-style
- Use `filepath.ToSlash()` or `filepath.FromSlash()` as appropriate
- Test on Windows CI with paths containing spaces and special chars

---

### 3.2 JSON-RPC Protocol Adapter

**Package:** `adapters/jsonrpc/`

**Implements:** `ports.ProtocolHandler`

**Responsibility:** Manages control protocol state, request routing, timeout handling, and concurrent request tracking.

#### 3.2.1 Request State Management - Failure Modes

**What Can Go Wrong:**

1. **Request ID collision** - Duplicate IDs cause response routing to wrong caller
2. **Memory leak from abandoned requests** - Pending requests never cleaned up if response never arrives
3. **Response for unknown request** - CLI sends response for request we didn't track
4. **Concurrent map access** - Race conditions when routing responses vs creating new requests
5. **Channel blocking** - Result channel full, blocking response handler
6. **Context cancellation not propagated** - Request cancelled but remains in pending map

**State Tracking Strategy:**

```go
type ProtocolAdapter struct {
    transport      ports.Transport
    pendingReqs    map[string]chan result
    requestCounter int
    mu             sync.Mutex

    // Observability
    metrics        ProtocolMetrics
    maxPending     int  // Track high-water mark for capacity planning
}

// Cleanup abandoned requests periodically
func (a *ProtocolAdapter) cleanupAbandonedRequests(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            a.mu.Lock()
            // Close channels for requests older than 5 minutes
            // (60s timeout + 4min grace for hung CLI)
            orphaned := 0
            for id, ch := range a.pendingReqs {
                // Extract timestamp from request ID format
                if isOlderThan(id, 5*time.Minute) {
                    close(ch)
                    delete(a.pendingReqs, id)
                    orphaned++
                }
            }
            if orphaned > 0 {
                a.metrics.RecordOrphanedRequests(orphaned)
            }
            a.mu.Unlock()
        }
    }
}
```

**Request ID Generation:**

```go
// Format: req_{counter}_{timestamp}_{randomHex}
// Timestamp enables age-based cleanup
func (a *ProtocolAdapter) generateRequestID() string {
    a.mu.Lock()
    a.requestCounter++
    counter := a.requestCounter
    a.mu.Unlock()

    timestamp := time.Now().Unix()
    random := randomHex(4)

    return fmt.Sprintf("req_%d_%d_%s", counter, timestamp, random)
}

// Extract timestamp for cleanup logic
func extractTimestamp(requestID string) (time.Time, error) {
    parts := strings.Split(requestID, "_")
    if len(parts) < 3 {
        return time.Time{}, fmt.Errorf("invalid request ID format")
    }

    ts, err := strconv.ParseInt(parts[2], 10, 64)
    if err != nil {
        return time.Time{}, err
    }

    return time.Unix(ts, 0), nil
}
```

#### 3.2.2 Timeout Handling - Production Concerns

**Timeout Hierarchy:**

- **Default control request timeout:** 60s (covers hooks, permissions, MCP routing)
- **Hook execution timeout:** 30s (prevent infinite user code)
- **Permission prompt timeout:** 120s (user may be slow to respond)
- **MCP server timeout:** 45s (external server may be slow)

**Timeout Strategy:**

```go
func (a *ProtocolAdapter) SendControlRequest(
    ctx context.Context,
    req map[string]any,
) (map[string]any, error) {
    requestID := a.generateRequestID()
    resCh := make(chan result, 1)  // Buffered to prevent goroutine leak

    a.mu.Lock()
    a.pendingReqs[requestID] = resCh
    a.mu.Unlock()

    // Cleanup on all exit paths
    defer func() {
        a.mu.Lock()
        delete(a.pendingReqs, requestID)
        a.mu.Unlock()
    }()

    // Send request to transport
    if err := a.sendRequest(ctx, requestID, req); err != nil {
        return nil, err
    }

    // Determine timeout based on request subtype
    timeout := a.getTimeoutForSubtype(req["subtype"].(string))
    timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    select {
    case <-timeoutCtx.Done():
        if timeoutCtx.Err() == context.DeadlineExceeded {
            a.metrics.RecordTimeout(req["subtype"].(string))
            return nil, fmt.Errorf("control request timeout after %s: %s",
                timeout, req["subtype"])
        }
        return nil, timeoutCtx.Err()  // Parent context cancelled

    case res := <-resCh:
        if res.err != nil {
            a.metrics.RecordError(req["subtype"].(string))
            return nil, res.err
        }
        a.metrics.RecordSuccess(req["subtype"].(string))
        return res.data, nil
    }
}

func (a *ProtocolAdapter) getTimeoutForSubtype(subtype string) time.Duration {
    switch subtype {
    case "hook_callback":
        return 30 * time.Second
    case "can_use_tool":
        return 120 * time.Second  // User permission prompt
    case "mcp_message":
        return 45 * time.Second
    default:
        return 60 * time.Second
    }
}
```

**Observability for Timeouts:**

- Log timeout events with request details (subtype, input, duration)
- Track timeout rate per subtype (high timeout rate indicates configuration issue)
- Include timeout context in error messages for user debugging

#### 3.2.3 Message Routing - Race Conditions & Error Cases

**What Can Go Wrong:**

1. **Response arrives before request registered** - Race between send and router startup
2. **Multiple responses for same request** - CLI bug or protocol confusion
3. **Malformed control messages** - Missing required fields cause panic in type assertions
4. **Router goroutine exits prematurely** - Transport error kills router, requests hang forever
5. **Channel send blocks** - Sending to msgCh/errCh blocks if consumer slow

**Router Resilience:**

```go
func (a *ProtocolAdapter) StartMessageRouter(
    ctx context.Context,
    msgCh chan<- map[string]any,
    errCh chan<- error,
    deps ControlDependencies,
) error {
    // Ensure router keeps running despite individual message errors
    go func() {
        defer func() {
            if r := recover(); r != nil {
                a.metrics.RecordRouterPanic(fmt.Sprintf("%v", r))
                // Send error to notify caller of router failure
                select {
                case errCh <- fmt.Errorf("router panicked: %v", r):
                default:
                }
            }
        }()

        transportMsgCh, transportErrCh := a.transport.ReadMessages(ctx)
        for {
            select {
            case <-ctx.Done():
                a.cleanupAllPendingRequests()
                return

            case msg, ok := <-transportMsgCh:
                if !ok {
                    a.cleanupAllPendingRequests()
                    return
                }

                // Recover from panic in message handling
                func() {
                    defer func() {
                        if r := recover(); r != nil {
                            a.metrics.RecordMessagePanic(fmt.Sprintf("%v", r))
                        }
                    }()
                    a.routeMessage(ctx, msg, msgCh, deps)
                }()

            case err := <-transportErrCh:
                select {
                case errCh <- err:
                case <-ctx.Done():
                }
                return
            }
        }
    }()

    return nil
}
```

**Safe Field Extraction:**

```go
// Safely extract fields with defaults/validation
func getStringField(msg map[string]any, field string) (string, error) {
    val, ok := msg[field]
    if !ok {
        return "", fmt.Errorf("missing required field: %s", field)
    }

    str, ok := val.(string)
    if !ok {
        return "", fmt.Errorf("field %s must be string, got %T", field, val)
    }

    return str, nil
}

func (a *ProtocolAdapter) routeControlResponse(msg map[string]any) {
    // Validate structure before accessing
    response, ok := msg["response"].(map[string]any)
    if !ok {
        a.metrics.RecordMalformedMessage("control_response")
        return
    }

    requestID, err := getStringField(response, "request_id")
    if err != nil {
        a.metrics.RecordMalformedMessage("control_response")
        return
    }

    a.mu.Lock()
    ch, exists := a.pendingReqs[requestID]
    delete(a.pendingReqs, requestID)
    a.mu.Unlock()

    if !exists {
        a.metrics.RecordUnknownResponse(requestID)
        return
    }

    // Send with timeout to prevent blocking on full channel
    select {
    case ch <- a.parseResponse(response):
    case <-time.After(1 * time.Second):
        a.metrics.RecordBlockedResponseSend(requestID)
    }
}
```

#### 3.2.4 Error Propagation & Diagnostics

**Error Categories:**

1. **Protocol errors:** Malformed messages, unknown request types
2. **Timeout errors:** Request exceeded deadline
3. **Transport errors:** Subprocess died, I/O error
4. **Handler errors:** Hook/permission/MCP handler returned error

**Structured Error Types:**

```go
type ProtocolError struct {
    Category    string              // "protocol", "timeout", "transport", "handler"
    RequestID   string
    Subtype     string
    Cause       error
    RequestData map[string]any      // For debugging
}

func (e *ProtocolError) Error() string {
    return fmt.Sprintf("protocol %s error [%s/%s]: %v",
        e.Category, e.RequestID, e.Subtype, e.Cause)
}
```

**Observability Hooks:**

```go
type ProtocolMetrics interface {
    RecordRequest(subtype string)
    RecordSuccess(subtype string)
    RecordError(subtype string)
    RecordTimeout(subtype string)
    RecordMalformedMessage(msgType string)
    RecordUnknownResponse(requestID string)
    RecordPendingRequests(count int)  // Gauge
    RecordOrphanedRequests(count int)
}
```

---

### 3.3 Message Parser Adapter

**Package:** `adapters/parse/`

**Implements:** `ports.MessageParser`

**Responsibility:** Converts raw JSON maps from transport into typed domain message structures.

#### 3.3.1 Parsing Failures - Robustness Strategy

**What Can Go Wrong:**

1. **Type assertion panics** - Field exists but has unexpected type (string vs number, etc.)
2. **Missing required fields** - CLI sends incomplete message
3. **Unknown message types** - CLI introduces new message type, SDK doesn't recognize
4. **Malformed JSON** - Already parsed by transport, but structure invalid
5. **Encoding issues** - UTF-8 problems in text fields
6. **Array element type mismatches** - Content blocks have unexpected structure

**Safe Parsing Pattern:**

```go
// Never panic on type assertions - always check and return error
func getStringField(data map[string]any, field string, required bool) (string, error) {
    val, ok := data[field]
    if !ok {
        if required {
            return "", fmt.Errorf("missing required field: %s", field)
        }
        return "", nil  // Optional field missing
    }

    str, ok := val.(string)
    if !ok {
        return "", fmt.Errorf("field %s must be string, got %T", field, val)
    }

    return str, nil
}

func getOptionalString(data map[string]any, field string) *string {
    if str, err := getStringField(data, field, false); err == nil && str != "" {
        return &str
    }
    return nil
}
```

**Unknown Message Type Handling:**

```go
func (a *Adapter) Parse(data map[string]any) (messages.Message, error) {
    msgType, err := getStringField(data, "type", true)
    if err != nil {
        return nil, err
    }

    switch msgType {
    case "user", "assistant", "system", "result", "stream_event":
        return a.parseKnownType(msgType, data)
    default:
        // Don't fail - wrap in UnknownMessage for forward compatibility
        a.metrics.RecordUnknownMessageType(msgType)
        return &messages.UnknownMessage{
            Type: msgType,
            Raw:  data,
        }, nil
    }
}
```

#### 3.3.2 Content Block Parsing - Union Type Handling

**Challenge:** Content can be string or array of blocks, blocks can be text/thinking/tool_use/tool_result.

**Type-Safe Approach:**

```go
// Parse content with fallback for unknown block types
func parseContentBlocks(contentArr []any) ([]messages.ContentBlock, error) {
    blocks := make([]messages.ContentBlock, 0, len(contentArr))

    for i, item := range contentArr {
        blockMap, ok := item.(map[string]any)
        if !ok {
            return nil, fmt.Errorf("content block %d must be object, got %T", i, item)
        }

        blockType, err := getStringField(blockMap, "type", true)
        if err != nil {
            return nil, fmt.Errorf("content block %d: %w", i, err)
        }

        var block messages.ContentBlock
        var parseErr error

        switch blockType {
        case "text":
            block, parseErr = parseTextBlock(blockMap)
        case "thinking":
            block, parseErr = parseThinkingBlock(blockMap)
        case "tool_use":
            block, parseErr = parseToolUseBlock(blockMap)
        case "tool_result":
            block, parseErr = parseToolResultBlock(blockMap)
        default:
            // Forward compatibility: preserve unknown blocks as raw data
            block = messages.UnknownContentBlock{
                Type: blockType,
                Raw:  blockMap,
            }
        }

        if parseErr != nil {
            return nil, fmt.Errorf("parse %s block at index %d: %w", blockType, i, parseErr)
        }

        blocks = append(blocks, block)
    }

    return blocks, nil
}
```

#### 3.3.3 Error Context - Debugging Support

**Attach Full Context to Parse Errors:**

```go
type ParseError struct {
    MessageType string
    Field       string
    Cause       error
    RawData     map[string]any  // Include for debugging (truncate if large)
}

func (e *ParseError) Error() string {
    // Truncate raw data for readability
    rawStr := fmt.Sprintf("%v", e.RawData)
    if len(rawStr) > 200 {
        rawStr = rawStr[:200] + "..."
    }

    return fmt.Sprintf("parse %s message field %s: %v (raw: %s)",
        e.MessageType, e.Field, e.Cause, rawStr)
}
```

**Observability:**

```go
type ParserMetrics interface {
    RecordParse(messageType string, success bool)
    RecordUnknownMessageType(msgType string)
    RecordUnknownBlockType(blockType string)
    RecordParseError(messageType string, field string)
}
```

---

### 3.4 File Organization - Linting Compliance

**Current state:** Massive single-file adapters violate 175-line limit.

**Required decomposition:**

#### CLI Adapter Package (`adapters/cli/`)

1. `transport.go` - Adapter struct, interface compliance, NewAdapter (60 lines)
2. `discovery.go` - CLI discovery with platform-specific logic (80 lines)
3. `connect.go` - Process startup, pipe setup (70 lines)
4. `command.go` - Command building, option handling (85 lines)
5. `io.go` - ReadMessages, Write, buffer management (90 lines)
6. `health.go` - Process monitoring, graceful shutdown (75 lines)
7. `errors.go` - CLIError type, diagnostic formatting (50 lines)

**Total:** ~510 lines across 7 files (vs 400+ in one file)

#### JSON-RPC Adapter Package (`adapters/jsonrpc/`)

1. `protocol.go` - Adapter struct, interface, NewAdapter (50 lines)
2. `requests.go` - SendControlRequest, request ID generation (80 lines)
3. `routing.go` - StartMessageRouter, message partitioning (90 lines)
4. `handlers.go` - HandleControlRequest, per-subtype routing (85 lines)
5. `cleanup.go` - Abandoned request cleanup, state management (60 lines)
6. `errors.go` - ProtocolError type, error categories (40 lines)

**Total:** ~405 lines across 6 files

#### Parser Adapter Package (`adapters/parse/`)

1. `parser.go` - Adapter struct, Parse dispatcher (60 lines)
2. `user.go` - User message parsing (70 lines)
3. `assistant.go` - Assistant message parsing (80 lines)
4. `system.go` - System message parsing (50 lines)
5. `result.go` - Result message parsing (90 lines)
6. `stream.go` - Stream event parsing (60 lines)
7. `blocks.go` - Content block parsing (150 lines - complex unions)
8. `helpers.go` - Field extraction helpers (60 lines)
9. `errors.go` - ParseError type (40 lines)

**Total:** ~660 lines across 9 files (vs 1100+ in one file)

---

## Implementation Sequence

**Phase 3.1: CLI Transport** (Week 1)

- Day 1-2: Discovery + platform testing
- Day 3-4: Process lifecycle + health monitoring
- Day 5: I/O streaming + buffer management
- Day 6-7: Error handling + observability hooks

**Acceptance Criteria:**
- CLI discovered on Windows, macOS, Linux
- Process terminates gracefully (no zombies)
- Large messages (>1MB) handled without saturation
- Timeout/error context included in all errors

**Phase 3.2: JSON-RPC Protocol** (Week 2)

- Day 1-2: State management + concurrent safety
- Day 3-4: Message routing + request lifecycle
- Day 5: Timeout handling per subtype
- Day 6-7: Cleanup logic + leak prevention

**Acceptance Criteria:**
- No request ID collisions under concurrent load
- Abandoned requests cleaned up within 5 minutes
- Timeout errors distinguish deadline vs cancellation
- Router resilient to malformed messages (no panics)

**Phase 3.3: Message Parser** (Week 2-3)

- Day 1-2: Safe field extraction helpers
- Day 3-4: Per-message-type parsers
- Day 5-6: Content block union handling
- Day 7: Unknown type forward compatibility

**Acceptance Criteria:**
- No panics on type assertion failures
- Unknown message/block types wrapped, not rejected
- Parse errors include full diagnostic context
- Tests cover all message types from Python SDK

---

## Testing Strategy

### Failure Mode Tests (Priority: Critical)

**CLI Transport:**
- CLI not found (expect clear error with discovery trace)
- CLI exits immediately (expect process health error)
- Stdout saturation (expect buffer overflow error)
- Stdin write to dead process (expect broken pipe error)
- Graceful shutdown timeout (expect SIGKILL after 15s)

**JSON-RPC Protocol:**
- Request timeout (expect timeout error with subtype)
- Response for unknown request (expect metric, no crash)
- Concurrent request storm (expect no ID collisions)
- Router goroutine panic (expect error propagated to caller)
- Abandoned requests (expect cleanup after 5min)

**Message Parser:**
- Missing required field (expect ParseError with field name)
- Wrong type for field (expect ParseError with type info)
- Unknown message type (expect UnknownMessage wrapper)
- Unknown block type (expect UnknownContentBlock wrapper)
- Malformed content array (expect error with index)

### Platform-Specific Tests

**Windows:**
- CLI discovery with .exe/.cmd/.bat extensions
- Paths with spaces and backslashes
- SIGTERM/SIGKILL equivalents (process termination)

**macOS/Linux:**
- Permission denied on executable
- Symlink resolution
- Signal propagation to child process

---

## Observability Checklist

**Required Metrics:**
- CLI discovery duration and success rate
- Process lifetime and exit codes
- Message throughput (msgs/sec, bytes/sec)
- Control request latency by subtype
- Timeout rate by subtype
- Parse error rate by message type
- Unknown message/block type occurrences

**Required Logs:**
- CLI path, version, PID at startup
- Full command-line for reproduction
- Health warnings (no activity, stderr saturation)
- All errors with structured diagnostic fields
- Process exit with runtime and message count

**Suggested Trace Points:**
- SendControlRequest entry/exit
- Message routing decision (control vs SDK)
- Parse entry/exit per message type

---

## Open Questions for Phase 4

1. **Metrics interface:** Should adapters accept optional metrics interface, or use global registry?
2. **Health monitoring:** Should health checks be opt-in or always enabled?
3. **Buffer limits:** Should max buffer size be per-message or total process memory?
4. **Timeout configuration:** Should users be able to override per-subtype timeouts?
5. **Unknown message handling:** Should UnknownMessage be surfaced to public API or filtered internally?

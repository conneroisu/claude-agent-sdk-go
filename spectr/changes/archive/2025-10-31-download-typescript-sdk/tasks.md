# Tasks: Download TypeScript SDK Script

## Implementation Order

### Phase 1: Script Foundation (REQ-001, REQ-002)
- [x] **Task 1.1**: Create `scripts/` directory if it doesn't exist
  - **Validation**: Directory exists at repository root
  - **Files**: `scripts/` directory created

- [x] **Task 1.2**: Create `scripts/download-ts-sdk.sh` with basic structure
  - **Validation**: File exists and has shebang (`#!/usr/bin/env bash`)
  - **Files**: `scripts/download-ts-sdk.sh`
  - **Dependencies**: Task 1.1

- [x] **Task 1.3**: Add execute permissions to script
  - **Validation**: `test -x scripts/download-ts-sdk.sh` returns 0
  - **Files**: `scripts/download-ts-sdk.sh` (permissions modified)
  - **Dependencies**: Task 1.2

- [x] **Task 1.4**: Implement directory creation logic
  - **Validation**: Script creates `.claude/contexts/` if missing
  - **Files**: `scripts/download-ts-sdk.sh` (logic added)
  - **Dependencies**: Task 1.3

- [x] **Task 1.5**: Implement npm install command
  - **Validation**: Script downloads SDK to `.claude/contexts/claude-agent-sdk-ts`
  - **Files**: `scripts/download-ts-sdk.sh` (npm logic added)
  - **Dependencies**: Task 1.4
  - **Command**: `npm install --prefix .claude/contexts/claude-agent-sdk-ts @anthropic-ai/claude-agent-sdk`

### Phase 2: Error Handling (REQ-004, REQ-005)
- [x] **Task 2.1**: Add npm availability check
  - **Validation**: Script exits with error if npm not found
  - **Files**: `scripts/download-ts-sdk.sh` (dependency check added)
  - **Dependencies**: Task 1.5
  - **Test**: Run script without npm in PATH

- [x] **Task 2.2**: Add error handling for npm install failures
  - **Validation**: Script exits cleanly on npm errors with status 1
  - **Files**: `scripts/download-ts-sdk.sh` (error handling added)
  - **Dependencies**: Task 2.1
  - **Test**: Mock npm failure (permissions, network)

- [x] **Task 2.3**: Add error handling for directory creation
  - **Validation**: Script reports directory creation failures
  - **Files**: `scripts/download-ts-sdk.sh` (mkdir error handling)
  - **Dependencies**: Task 2.2
  - **Test**: Mock directory creation failure

### Phase 3: Idempotency (REQ-003)
- [x] **Task 3.1**: Add check for existing installation
  - **Validation**: Script detects existing SDK installation
  - **Files**: `scripts/download-ts-sdk.sh` (existence check)
  - **Dependencies**: Task 2.3

- [x] **Task 3.2**: Implement update logic for existing installation
  - **Validation**: Script updates SDK when version is outdated
  - **Files**: `scripts/download-ts-sdk.sh` (update logic)
  - **Dependencies**: Task 3.1
  - **Command**: `npm update --prefix .claude/contexts/claude-agent-sdk-ts @anthropic-ai/claude-agent-sdk`

### Phase 4: User Experience (REQ-007)
- [x] **Task 4.1**: Add progress messages
  - **Validation**: Script prints clear status at each step
  - **Files**: `scripts/download-ts-sdk.sh` (logging added)
  - **Dependencies**: Task 3.2
  - **Messages**:
    - "Checking for npm..."
    - "Creating target directory..."
    - "Downloading @anthropic-ai/claude-agent-sdk..."
    - "TypeScript SDK successfully installed to .claude/contexts/claude-agent-sdk-ts"

- [x] **Task 4.2**: Add helpful error messages
  - **Validation**: Error messages include resolution steps
  - **Files**: `scripts/download-ts-sdk.sh` (error messages improved)
  - **Dependencies**: Task 4.1

### Phase 5: Git Integration (REQ-006)
- [x] **Task 5.1**: Add `.claude/contexts/` to `.gitignore`
  - **Validation**: Downloaded SDK not tracked by git
  - **Files**: `.gitignore` (entry added)
  - **Dependencies**: None (can run in parallel with other tasks)
  - **Check**: Verify `.claude/contexts/` not already ignored

### Phase 6: Documentation (REQ-008)
- [x] **Task 6.1**: Add script documentation to README or docs
  - **Validation**: Developers know about the download script
  - **Files**: `README.md` or `docs/development.md`
  - **Dependencies**: All implementation tasks complete
  - **Content**:
    - How to run the script
    - What it does
    - Platform requirements (npm)
    - Windows users should use Git Bash or WSL

### Phase 7: Testing
- [x] **Task 7.1**: Test fresh installation
  - **Validation**: Script works on clean repository
  - **Dependencies**: All implementation complete
  - **Test Steps**:
    1. Remove `.claude/contexts/` if exists
    2. Run `./scripts/download-ts-sdk.sh`
    3. Verify SDK installed correctly
    4. Check git status shows ignored directory

- [x] **Task 7.2**: Test idempotency
  - **Validation**: Script handles existing installation
  - **Dependencies**: Task 7.1
  - **Test Steps**:
    1. Run script twice in succession
    2. Verify no errors on second run
    3. Verify SDK is at latest version

- [x] **Task 7.3**: Test error scenarios
  - **Validation**: Error handling works correctly
  - **Dependencies**: Task 7.2
  - **Test Scenarios**:
    - Run without npm in PATH
    - Run with readonly parent directory
    - Mock network failure

## Parallel Work Opportunities
- Task 5.1 (gitignore) can be done independently
- Task 6.1 (documentation) can be drafted while implementation is in progress

## Dependencies Summary
- **External**: npm must be installed (documented requirement)
- **Internal**: None (new script, no impact on existing code)

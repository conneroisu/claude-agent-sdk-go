# Proposal: Download TypeScript SDK Script

## Change ID
`download-typescript-sdk`

## Status
Proposed

## Summary
Add a bash script to download and maintain the latest version of the Anthropic TypeScript SDK (`@anthropic-ai/claude-agent-sdk`) to `.claude/contexts/claude-agent-sdk-ts` for reference and comparison purposes.

## Motivation
The Go SDK needs to maintain parity with the TypeScript SDK (as noted in `openspec/project.md`). Having the TypeScript SDK source code locally available enables:

1. **API Parity Verification**: Developers can quickly reference TypeScript implementation details when implementing Go equivalents
2. **Documentation**: TypeScript SDK serves as reference documentation for features and patterns
3. **Testing**: Allows comparing behavior between SDKs
4. **Development Workflow**: Automated download ensures developers always have the latest version without manual steps

## Scope
This change introduces a single capability:

- **typescript-sdk-download**: Bash script to download/update the TypeScript SDK using npm

## Impact
- **Users**: No impact (development tooling only)
- **Developers**: Simplified workflow for SDK comparison and parity checking
- **CI/CD**: Could be integrated into development setup scripts
- **Dependencies**: Requires Node.js and npm to be installed

## Non-Goals
- This script does NOT install the TypeScript SDK for runtime use
- This script does NOT create bindings between Go and TypeScript
- This script does NOT automatically sync API changes (manual comparison still required)

## Alternatives Considered
1. **Git submodule**: Would lock to specific version, requires manual updates, adds git complexity
2. **Manual download**: Error-prone, inconsistent across developers
3. **GitHub API download**: More complex, npm is simpler for a Node.js package

## Related Changes
None - this is a standalone development tool addition.

## Success Criteria
- Script successfully downloads TypeScript SDK to `.claude/contexts/claude-agent-sdk-ts`
- Script is idempotent (safe to run multiple times)
- Script updates existing installation if present
- Script fails gracefully with helpful error messages if npm is not available

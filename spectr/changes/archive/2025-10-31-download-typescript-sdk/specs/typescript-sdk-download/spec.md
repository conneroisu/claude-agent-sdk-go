# Spec: TypeScript SDK Download Script

## Overview
A bash script that downloads and maintains the Anthropic TypeScript SDK (`@anthropic-ai/claude-agent-sdk`) from npm to a local directory for development reference.

---

## ADDED Requirements

### Requirement: Script Location and Execution
The download script SHALL be located at `scripts/download-ts-sdk.sh` and SHALL be executable, downloading the TypeScript SDK to `.claude/contexts/claude-agent-sdk-ts`.

#### Scenario: Developer runs the download script
- **GIVEN** the repository is cloned
- **AND** the developer has execute permissions
- **WHEN** they run `./scripts/download-ts-sdk.sh`
- **THEN** the script executes successfully
- **AND** downloads the TypeScript SDK to `.claude/contexts/claude-agent-sdk-ts`

---

### Requirement: NPM-based Download
The script SHALL use npm to install the `@anthropic-ai/claude-agent-sdk` package to the specified directory.

#### Scenario: Script downloads latest SDK version
- **GIVEN** npm is installed and available in PATH
- **AND** the target directory `.claude/contexts/claude-agent-sdk-ts` does not exist
- **WHEN** the script executes
- **THEN** it runs `npm install @anthropic-ai/claude-agent-sdk` to the target directory
- **AND** the latest version is downloaded

#### Scenario: Script downloads to correct location
- **GIVEN** the script is executed from any directory
- **WHEN** the download completes
- **THEN** the SDK is installed to `.claude/contexts/claude-agent-sdk-ts`
- **AND** the path is relative to the repository root

---

### Requirement: Idempotent Updates
The script SHALL be idempotent, updating existing installations without errors.

#### Scenario: Script updates existing installation
- **GIVEN** the TypeScript SDK is already installed in `.claude/contexts/claude-agent-sdk-ts`
- **AND** a newer version is available
- **WHEN** the script executes
- **THEN** it updates the existing installation using `npm update`
- **AND** does not create duplicate installations

#### Scenario: Script handles no updates available
- **GIVEN** the TypeScript SDK is already at the latest version
- **WHEN** the script executes
- **THEN** npm reports no updates needed
- **AND** the script exits successfully with status 0

---

### Requirement: Dependency Verification
The script SHALL verify npm is installed before attempting download.

#### Scenario: npm is not installed
- **GIVEN** npm is not available in PATH
- **WHEN** the script executes
- **THEN** it prints a clear error message: "Error: npm is required but not installed"
- **AND** exits with status 1
- **AND** provides installation instructions for common platforms

#### Scenario: npm is installed
- **GIVEN** npm is available in PATH
- **WHEN** the script checks dependencies
- **THEN** it proceeds with the download
- **AND** does not print dependency warnings

---

### Requirement: Error Handling and Reporting
The script SHALL handle common failure scenarios with helpful error messages.

#### Scenario: npm install fails
- **GIVEN** npm is installed
- **BUT** the npm install command fails (network error, permissions, etc.)
- **WHEN** the script detects the failure
- **THEN** it prints the npm error output
- **AND** exits with status 1
- **AND** does not leave partial installations

#### Scenario: Directory creation fails
- **GIVEN** the `.claude/contexts` directory cannot be created (permissions, disk space, etc.)
- **WHEN** the script attempts to create it
- **THEN** it prints a clear error message about the failure
- **AND** exits with status 1

---

### Requirement: Gitignore Integration
The downloaded SDK SHALL be ignored by git to avoid committing dependencies.

#### Scenario: Downloaded SDK is gitignored
- **GIVEN** the TypeScript SDK is downloaded to `.claude/contexts/claude-agent-sdk-ts`
- **WHEN** a developer runs `git status`
- **THEN** the directory does not appear as untracked
- **AND** `.gitignore` contains an entry for `.claude/contexts/`

---

### Requirement: Script Output and Logging
The script SHALL provide clear, concise output about what it is doing.

#### Scenario: Script shows progress
- **GIVEN** the script is executed
- **WHEN** it performs each major step
- **THEN** it prints status messages:
  - "Checking for npm..."
  - "Creating target directory..."
  - "Downloading @anthropic-ai/claude-agent-sdk..."
  - "TypeScript SDK successfully installed to .claude/contexts/claude-agent-sdk-ts"

#### Scenario: Script runs quietly on success
- **GIVEN** the script completes successfully
- **WHEN** all operations succeed
- **THEN** output is concise (4-5 lines maximum)
- **AND** verbose details are only shown on errors

---

### Requirement: Cross-platform Compatibility
The script SHALL work on common development platforms (Linux, macOS) with guidance for Windows users.

#### Scenario: Script runs on Linux
- **GIVEN** a Linux development environment
- **WHEN** the script executes
- **THEN** all commands work correctly

#### Scenario: Script runs on macOS
- **GIVEN** a macOS development environment
- **WHEN** the script executes
- **THEN** all commands work correctly

#### Scenario: Script provides guidance for Windows
- **GIVEN** a Windows development environment
- **WHEN** a developer tries to run the bash script
- **THEN** the README or script comments suggest using Git Bash or WSL

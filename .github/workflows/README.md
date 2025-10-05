# GitHub Actions Workflows

This directory contains CI/CD workflows for the Claude Agent SDK for Go.

## Workflows

### lint.yml
- **Trigger:** Push/PR to main
- **Purpose:** Run golangci-lint on all Go code
- **Fail Condition:** Any linting violation

### test.yml
- **Trigger:** Push/PR to main
- **Purpose:** Run tests on Go 1.22 and 1.23
- **Coverage:** Upload to Codecov
- **Matrix:** Multiple Go versions

### release.yml
- **Trigger:** Version tags (v*.*.*)
- **Purpose:** Create GitHub releases
- **Steps:**
  1. Run full test suite
  2. Run linter
  3. Extract release notes from CHANGELOG.md
  4. Create GitHub release

### security.yml
- **Trigger:** Push/PR to main, weekly schedule
- **Purpose:** Security scanning
- **Tools:**
  - CodeQL for static analysis
  - govulncheck for vulnerability detection

## Release Process

1. Update VERSION file
2. Update CHANGELOG.md with release notes
3. Run `./scripts/release.sh v0.1.0`
4. Script will:
   - Validate version format
   - Run tests and linter
   - Create git tag
   - Push to trigger release workflow

## Security

- Dependabot monitors dependencies weekly
- CodeQL scans code for vulnerabilities
- govulncheck detects known Go vulnerabilities
- All workflows require passing status checks

## Phase 7: Publishing & CI/CD

### 7.1 Go Module Setup
Priority: Critical

**Actionable Tasks:**

1. **Initialize Go module with semantic versioning:**
   ```bash
   go mod init github.com/conneroisu/claude-agent-sdk-go
   go mod tidy
   ```

2. **Configure go.mod for Go 1.25+ compatibility:**
   ```go
   module github.com/conneroisu/claude-agent-sdk-go

   go 1.25

   require (
       github.com/modelcontextprotocol/go-sdk v0.1.0
   )
   ```

3. **Document module versioning strategy:**
   - Use semantic versioning (MAJOR.MINOR.PATCH)
   - Breaking changes increment MAJOR
   - New features increment MINOR
   - Bug fixes increment PATCH
   - Pre-releases use `-alpha`, `-beta`, `-rc` suffixes

4. **Create VERSION file tracking current version:**
   ```
   0.1.0-alpha
   ```

### 7.2 CI/CD Pipeline
Priority: High

**Actionable Tasks:**

1. **Set up GitHub Actions workflow for linting:**
   - Create `.github/workflows/lint.yml`
   - Run `golangci-lint run` on every push/PR
   - Fail CI on ANY linting violation
   - Use `golangci/golangci-lint-action@v4` with timeout 5m

2. **Set up GitHub Actions workflow for testing:**
   - Create `.github/workflows/test.yml`
   - Run tests on multiple Go versions (1.25, 1.26)
   - Generate coverage reports
   - Fail CI if coverage drops below 80%
   - Upload coverage to codecov.io

3. **Set up GitHub Actions workflow for releases:**
   - Create `.github/workflows/release.yml`
   - Trigger on version tags (v*.*.*)
   - Run full test suite before release
   - Generate release notes from CHANGELOG.md
   - Create GitHub release with binaries

4. **Configure branch protection rules:**
   - Require status checks to pass before merging
   - Require 1 approval for PRs
   - Enforce linear history
   - Protect main branch from force pushes

### 7.3 Release Process
Priority: Medium

**Actionable Tasks:**

1. **Create CHANGELOG.md following Keep a Changelog format:**
   ```markdown
   # Changelog

   All notable changes to this project will be documented in this file.

   The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
   and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

   ## [Unreleased]

   ## [0.1.0] - 2025-XX-XX
   ### Added
   - Initial release with Query and Client APIs
   - Full control protocol support
   - Hooks, permissions, and MCP integration
   ```

2. **Document release checklist in CONTRIBUTING.md:**
   - Update VERSION file
   - Update CHANGELOG.md with release date
   - Create git tag: `git tag -a v0.1.0 -m "Release v0.1.0"`
   - Push tag: `git push origin v0.1.0`
   - Verify GitHub Actions release workflow completes
   - Verify module published to pkg.go.dev

3. **Create release automation script (scripts/release.sh):**
   ```bash
   #!/bin/bash
   set -e

   VERSION=$1
   if [ -z "$VERSION" ]; then
       echo "Usage: ./scripts/release.sh v0.1.0"
       exit 1
   fi

   # Validate version format
   if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-z]+)?$ ]]; then
       echo "Invalid version format. Use vMAJOR.MINOR.PATCH"
       exit 1
   fi

   # Update VERSION file
   echo "$VERSION" > VERSION

   # Ensure CHANGELOG updated
   if ! grep -q "## \[$VERSION\]" CHANGELOG.md; then
       echo "ERROR: Update CHANGELOG.md with [$VERSION] entry"
       exit 1
   fi

   # Run tests
   go test ./...

   # Run linter
   golangci-lint run

   # Create and push tag
   git add VERSION CHANGELOG.md
   git commit -m "Release $VERSION"
   git tag -a "$VERSION" -m "Release $VERSION"
   git push origin main "$VERSION"

   echo "Release $VERSION completed!"
   ```

4. **Update documentation with installation instructions:**
   - Add to README.md:
     ```markdown
     ## Installation

     ```bash
     go get github.com/conneroisu/claude-agent-sdk-go@latest
     ```

     Or pin to specific version:
     ```bash
     go get github.com/conneroisu/claude-agent-sdk-go@v0.1.0
     ```
     ```

5. **Create pkg.go.dev documentation:**
   - Ensure all public APIs have godoc comments
   - Include runnable examples in _test.go files
   - Add package-level documentation in doc.go
   - Verify documentation renders correctly on pkg.go.dev

---

## Linting Compliance Notes

### CI/CD Integration

**Critical: golangci-lint must be enforced in CI**

```yaml
# .github/workflows/lint.yml
name: Lint
on: [push, pull_request]

jobs:
  golangci:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
          args: --timeout=5m
          # This will fail CI on ANY linting violation
```

**Pre-commit hooks:**
```bash
#!/bin/bash
# .git/hooks/pre-commit
golangci-lint run --new-from-rev=HEAD~1
if [ $? -ne 0 ]; then
    echo "❌ Linting failed. Fix issues before committing."
    exit 1
fi
```

### Checklist

- [ ] golangci-lint runs in CI on every PR
- [ ] CI fails on ANY linting violation
- [ ] Pre-commit hooks configured
- [ ] Developer docs include linting setup instructions
- [ ] All linters from .golangci.yaml are enforced
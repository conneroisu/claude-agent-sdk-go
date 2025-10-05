## Phase 7: Publishing & CI/CD

### 7.1 Go Module Setup
Priority: Critical

**Actionable Tasks:**

1. **Initialize Go module with semantic versioning:**
   ```bash
   go mod init github.com/conneroisu/claude-agent-sdk-go
   go mod tidy
   ```

2. **Configure go.mod for Go 1.23+ compatibility:**
   ```go
   module github.com/conneroisu/claude-agent-sdk-go

   go 1.23

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
   - Run tests on multiple Go versions (1.22, 1.23)
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

   # Ensure we're on main branch
   CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
   if [ "$CURRENT_BRANCH" != "main" ]; then
       echo "ERROR: Must be on main branch to release"
       exit 1
   fi

   # Ensure working directory is clean
   if ! git diff-index --quiet HEAD --; then
       echo "ERROR: Working directory has uncommitted changes"
       exit 1
   fi

   # Ensure we're up to date with remote
   git fetch origin
   if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]; then
       echo "ERROR: Local main is not in sync with origin/main"
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
   echo "Running tests..."
   go test ./...

   # Run linter
   echo "Running linter..."
   golangci-lint run

   # Verify tag doesn't already exist
   if git rev-parse "$VERSION" >/dev/null 2>&1; then
       echo "ERROR: Tag $VERSION already exists"
       exit 1
   fi

   # Show what will be released and require confirmation
   echo ""
   echo "Ready to release $VERSION"
   echo "Changes since last release:"
   git log --oneline "$(git describe --tags --abbrev=0 2>/dev/null || echo '')..HEAD"
   echo ""
   read -p "Continue with release? (y/N) " -n 1 -r
   echo
   if [[ ! $REPLY =~ ^[Yy]$ ]]; then
       echo "Release cancelled"
       exit 1
   fi

   # Create commit and tag (but don't push yet)
   git add VERSION CHANGELOG.md
   git commit -m "Release $VERSION"
   git tag -a "$VERSION" -m "Release $VERSION"

   # Final confirmation before push
   echo ""
   echo "Commit and tag created locally"
   read -p "Push to origin? This will trigger the release workflow. (y/N) " -n 1 -r
   echo
   if [[ ! $REPLY =~ ^[Yy]$ ]]; then
       echo "To push manually: git push origin main $VERSION"
       exit 1
   fi

   # Push to remote (this will trigger GitHub Actions release workflow)
   git push origin main "$VERSION"

   echo "Release $VERSION completed!"
   echo "Monitor the release workflow at: https://github.com/conneroisu/claude-agent-sdk-go/actions"
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

### 7.4 Artifact Verification & Signing
Priority: High

**Actionable Tasks:**

1. **Generate and publish checksums for releases:**
   - Create SHA256 checksums for all release artifacts
   - Include checksums.txt in GitHub release assets
   - Document checksum verification in release notes
   - Example checksum generation:
     ```bash
     sha256sum claude-agent-sdk-go_*.tar.gz > checksums.txt
     ```

2. **Implement SLSA Build Provenance:**
   - Use SLSA GitHub Generator for Go projects
   - Generate verifiable build provenance (SLSA Level 3)
   - Publish provenance attestations with releases
   - Add SLSA badge to README.md
   - Configure GitHub Actions workflow:
     ```yaml
     # .github/workflows/release.yml
     - name: Generate SLSA provenance
       uses: slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v1.9.0
       with:
         attestation-name: provenance.intoto.jsonl
     ```

3. **Sign release tags with GPG:**
   - Configure Git to sign tags by default
   - Store GPG signing key as GitHub secret
   - Verify all release tags are signed
   - Document signature verification process:
     ```bash
     git tag -v v0.1.0  # Verify tag signature
     ```

4. **Enable Go module checksum database:**
   - Ensure module is indexed in sum.golang.org
   - Verify checksum database entries after release
   - Document checksum verification for users:
     ```bash
     go get -v github.com/conneroisu/claude-agent-sdk-go@v0.1.0
     # Go automatically verifies checksums from sum.golang.org
     ```

5. **Implement Software Bill of Materials (SBOM):**
   - Generate SBOM for each release using syft or cyclonedx-gomod
   - Include SBOM in release artifacts
   - Support both SPDX and CycloneDX formats
   - Example SBOM generation:
     ```bash
     syft packages . -o spdx-json > sbom.spdx.json
     syft packages . -o cyclonedx-json > sbom.cyclonedx.json
     ```

6. **Configure Dependabot security alerts:**
   - Enable Dependabot for dependency updates
   - Configure `.github/dependabot.yml` for Go modules
   - Set up automatic PR creation for security patches
   - Require security review for dependency updates

7. **Implement release verification checklist:**
   - [ ] All tests pass on CI
   - [ ] Code coverage meets 80% threshold
   - [ ] All linters pass with zero violations
   - [ ] CHANGELOG.md updated with release notes
   - [ ] VERSION file updated
   - [ ] Git tag signed with GPG key
   - [ ] Checksums generated and verified
   - [ ] SLSA provenance attestation published
   - [ ] SBOM generated and included
   - [ ] Release notes include security advisories (if any)
   - [ ] Module appears on pkg.go.dev within 24 hours
   - [ ] Checksum verified in sum.golang.org

### 7.5 Security Best Practices
Priority: High

**Actionable Tasks:**

1. **Configure GitHub repository security settings:**
   - Enable private vulnerability reporting
   - Configure security policy (SECURITY.md)
   - Enable secret scanning
   - Enable push protection for secrets
   - Require signed commits (optional but recommended)

2. **Set up automated security scanning:**
   - Configure CodeQL analysis in GitHub Actions
   - Run govulncheck on every CI build
   - Fail CI on HIGH or CRITICAL vulnerabilities
   - Example GitHub Actions integration:
     ```yaml
     - name: Run govulncheck
       uses: golang/govulncheck-action@v1
       with:
         go-version-input: 1.23
         check-latest: true
     ```

3. **Implement least-privilege access controls:**
   - Use GitHub environments for release workflows
   - Require manual approval for production releases
   - Restrict who can push to main branch
   - Limit access to signing keys and secrets

4. **Document vulnerability disclosure process:**
   - Create SECURITY.md with reporting guidelines
   - Define response timeline (acknowledge within 48h)
   - Establish patch release process for vulnerabilities
   - Document CVE assignment process if needed

5. **Enable OpenSSF Scorecard:**
   - Add OpenSSF Scorecard GitHub Action
   - Monitor security best practice compliance
   - Publish scorecard results in README.md
   - Address scorecard recommendations iteratively

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
          go-version: '1.23'
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
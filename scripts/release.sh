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

#!/bin/bash
set -e

# Épurer- Release Script

echo "🚀 Épurer- Release Script"
echo ""

# Check if we're on main branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    echo "⚠️  Warning: Not on main branch (current: $CURRENT_BRANCH)"
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check for uncommitted changes
if [[ -n $(git status -s) ]]; then
    echo "❌ Error: You have uncommitted changes"
    git status -s
    exit 1
fi

# Get current version from git tag
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "📦 Current version: $CURRENT_VERSION"
echo ""

# Ask for new version
read -p "Enter new version (e.g., v1.0.1): " NEW_VERSION

if [[ ! $NEW_VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "❌ Error: Invalid version format (must be vX.Y.Z)"
    exit 1
fi

echo ""
echo "📝 Creating release $NEW_VERSION..."
echo ""

# Update CHANGELOG.md
DATE=$(date +%Y-%m-%d)
echo "Updating CHANGELOG.md..."

# Build
echo "🔨 Building binary..."
make clean
make build

# Test
echo "🧪 Running tests..."
make test

# Create git tag
echo "🏷️  Creating git tag $NEW_VERSION..."
git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

echo ""
echo "✅ Release $NEW_VERSION prepared!"
echo ""
echo "Next steps:"
echo "  1. Review changes: git show $NEW_VERSION"
echo "  2. Push to GitHub: git push origin main --tags"
echo "  3. Create GitHub release at: https://github.com/0SansNom/epurer/releases/new"
echo "  4. Run goreleaser: goreleaser release --clean"
echo ""

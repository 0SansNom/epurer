#!/bin/bash
set -euo pipefail

# Épurer release prep: tests, annotated tag, universal DMG.
# Usage: ./release.sh v2.0.0
# Then:  git push origin master --tags && make release

echo "🚀 Épurer release"
echo ""

CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "master" ]]; then
    echo "⚠️  Warning: Not on master branch (current: $CURRENT_BRANCH)"
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

if [[ -n $(git status -s) ]]; then
    echo "❌ Error: You have uncommitted changes"
    git status -s
    exit 1
fi

CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
echo "📦 Current version: $CURRENT_VERSION"
echo ""

NEW_VERSION="${1:-}"
if [[ -z "$NEW_VERSION" ]]; then
    read -p "Enter new version (e.g., v1.0.1): " NEW_VERSION
fi

if [[ ! $NEW_VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "❌ Error: Invalid version format (must be vX.Y.Z)"
    exit 1
fi

if git rev-parse "$NEW_VERSION" >/dev/null 2>&1; then
    echo "❌ Error: Tag $NEW_VERSION already exists"
    exit 1
fi

echo ""
echo "📝 Preparing release $NEW_VERSION..."
echo ""

if ! grep -q "^## \[${NEW_VERSION#v}\]" CHANGELOG.md; then
    echo "❌ Error: CHANGELOG.md has no entry for ${NEW_VERSION#v}"
    exit 1
fi
if grep -q "^## \[${NEW_VERSION#v}\] - Unreleased" CHANGELOG.md; then
    echo "❌ Error: CHANGELOG.md still says Unreleased for ${NEW_VERSION#v} — set today's date first"
    exit 1
fi

echo "🔨 Building binary..."
make clean
make build

echo "🧪 Running tests..."
make test

echo "🏷️  Creating git tag $NEW_VERSION..."
git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

echo "💿 Building dist/Epurer.dmg..."
make dmg

echo ""
echo "✅ Release $NEW_VERSION prepared!"
echo ""
echo "Next steps:"
echo "  1. Review: git show $NEW_VERSION"
echo "  2. Push:   git push origin master --tags"
echo "  3. Publish (uses GitHub auth from gh):"
echo "       export GITHUB_TOKEN=\"\$(gh auth token)\""
echo "       make release"
echo ""

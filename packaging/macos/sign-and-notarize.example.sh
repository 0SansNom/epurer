#!/usr/bin/env bash
# Example only — fill in your Developer ID identity and Apple notarization profile.
# Policy: hardened runtime, timestamp, no privileged helper, entitlements from this folder.
#
# Typical flow after `make app-bundle`:
#   EPURER_SIGN_IDENTITY="Developer ID Application: …" \
#   EPURER_NOTARY_PROFILE="epurer-notary" \
#   ./packaging/macos/sign-and-notarize.example.sh dist/Épurer.app
set -euo pipefail

APP="${1:?usage: $0 /path/to/Épurer.app}"
IDENTITY="${EPURER_SIGN_IDENTITY:?set EPURER_SIGN_IDENTITY to 'Developer ID Application: ...'}"
PROFILE="${EPURER_NOTARY_PROFILE:?set EPURER_NOTARY_PROFILE to a notarytool keychain profile}"
ROOT="$(cd "$(dirname "$0")" && pwd)"

ENGINE="$APP/Contents/Resources/epurer-engine"
HOST="$APP/Contents/MacOS/Epurer"

if [[ ! -x "$ENGINE" ]]; then
  echo "missing embedded engine: $ENGINE" >&2
  exit 1
fi
if [[ ! -x "$HOST" ]]; then
  echo "missing app binary: $HOST" >&2
  exit 1
fi

# Sign nested engine first, then the host, then the bundle (hardened runtime).
codesign --force --options runtime --timestamp \
  --entitlements "$ROOT/epurer.entitlements" \
  --sign "$IDENTITY" \
  "$ENGINE"

codesign --force --options runtime --timestamp \
  --entitlements "$ROOT/epurer.entitlements" \
  --sign "$IDENTITY" \
  "$HOST"

codesign --force --deep --options runtime --timestamp \
  --entitlements "$ROOT/epurer.entitlements" \
  --sign "$IDENTITY" \
  "$APP"

codesign --verify --deep --strict --verbose=2 "$APP"
spctl --assess --type execute --verbose=4 "$APP" || true

ditto -c -k --keepParent "$APP" "${APP}.zip"
xcrun notarytool submit "${APP}.zip" --keychain-profile "$PROFILE" --wait
xcrun stapler staple "$APP"
echo "signed and notarized: $APP"

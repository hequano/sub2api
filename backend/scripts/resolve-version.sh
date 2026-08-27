#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
BACKEND_DIR="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"
REPO_DIR="$(CDPATH= cd -- "$BACKEND_DIR/.." && pwd)"
VERSION_FILE="$BACKEND_DIR/cmd/server/VERSION"
CUSTOM_VERSION_FILE="$BACKEND_DIR/cmd/server/CUSTOM_VERSION"

# Prefer the exact release tag when building from a tagged checkout so
# source builds from vX.Y.Z don't inherit the previous VERSION file value.
BASE_VERSION="$(tr -d '\r\n' < "$VERSION_FILE")"
if command -v git >/dev/null 2>&1; then
  TAG="$(
    git -C "$REPO_DIR" describe --tags --exact-match --match 'v[0-9]*' 2>/dev/null || \
    git -C "$REPO_DIR" describe --tags --exact-match --match '[0-9]*' 2>/dev/null || \
    true
  )"
  if [ -n "$TAG" ]; then
    BASE_VERSION="${TAG#v}"
  fi
fi

if [ -f "$CUSTOM_VERSION_FILE" ]; then
  CUSTOM_REV="$(tr -d '\r\n' < "$CUSTOM_VERSION_FILE" | sed 's/^[.]*//')"
  if [ -n "$CUSTOM_REV" ]; then
    printf '%s.%s\n' "$BASE_VERSION" "$CUSTOM_REV"
    exit 0
  fi
fi

printf '%s\n' "$BASE_VERSION"

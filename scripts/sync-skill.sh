#!/usr/bin/env bash
# Copy the canonical docs/site/skill.md into src/ so it can be embedded via
# //go:embed. The Go module root is src/ and docs/site/ sits above it, so embed
# cannot reach the canonical file directly — this copy step bridges the gap. The
# copy is committed so a clean `go build ./...` (which does not run this script)
# compiles; the drift-guard test in src/cmd/idea/skill_test.go keeps it
# byte-honest against docs/site/skill.md on every `go test`.
set -euo pipefail

# Run from the repo root regardless of caller CWD.
cd "$(dirname "$0")/.."

SRC="docs/site/skill.md"
DEST="src/cmd/idea/skill/skill.md"

mkdir -p "$(dirname "$DEST")"
cp -f "$SRC" "$DEST"
echo "synced skill bundle: ${DEST}"

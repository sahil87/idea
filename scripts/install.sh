#!/usr/bin/env bash
set -euo pipefail

./scripts/build.sh

DEST="${HOME}/.local/bin/idea"
mkdir -p "$(dirname "$DEST")"
cp -f ./bin/idea "$DEST"
echo "installed: $DEST"

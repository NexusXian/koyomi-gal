#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

SWAG_BIN="$(command -v swag || true)"
if [[ -z "$SWAG_BIN" && -x "$HOME/go/bin/swag" ]]; then
  SWAG_BIN="$HOME/go/bin/swag"
fi
if [[ -z "$SWAG_BIN" ]]; then
  echo "error: swag not found, install with: go install github.com/swaggo/swag/cmd/swag@latest" >&2
  exit 1
fi

"$SWAG_BIN" init -g cmd/server/main.go --parseInternal -o docs

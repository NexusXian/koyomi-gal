#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

SEARCH_DIRS="$(go list -f '{{.Dir}}' ./cmd/server ./internal/... ./pkg/... | paste -sd ',' -)"
go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g main.go -d "$SEARCH_DIRS" --parseInternal -o docs

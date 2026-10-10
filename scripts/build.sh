#!/usr/bin/env bash
# Builds both panels: Tailwind CSS first (needed by go:embed), then Go binaries.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

for app in panel admin; do
  echo "==> $app"
  (
    cd "$ROOT/$app"
    if [ ! -d node_modules ]; then
      npm install --silent
    fi
    npx tailwindcss -i static/src/input.css -o static/dist/output.css --minify
    go build -o bin/app ./cmd/...
  )
done

echo "Done. Binaries: panel/bin/app, admin/bin/app"

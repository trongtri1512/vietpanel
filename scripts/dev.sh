#!/usr/bin/env bash
# Builds and runs both panels locally for development.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

"$ROOT/scripts/build.sh"

cleanup() {
  pkill -f "vietpanel/.*/bin/app" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

(cd "$ROOT/panel" && ./bin/app > /tmp/vietpanel-panel.log 2>&1) &
(cd "$ROOT/admin" && ./bin/app > /tmp/vietpanel-admin.log 2>&1) &

sleep 2

echo "Panel:  http://localhost:2083/login  (demo / demo123)"
echo "Admin:  http://localhost:2087/login  (admin / admin123)"
echo "Logs:   /tmp/vietpanel-panel.log, /tmp/vietpanel-admin.log"
echo "Press Ctrl+C to stop both panels."

wait

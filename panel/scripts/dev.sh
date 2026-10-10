#!/bin/bash
# Dev runner for VietPanel panel on macOS (no sudo needed).
# Overrides the hardcoded /etc/openpanel Linux paths via env vars.
set -e
cd "$(dirname "$0")/.."

export VIETPANEL_CONFIG="$PWD/data/dev/conf/openpanel.config"
export VIETPANEL_SECRET="$PWD/data/dev/secret.key"
export VIETPANEL_TRANSLATIONS="/Users/trongtri/Desktop/Project/vietpanel/translations/"
export LISTEN_ADDR="${LISTEN_ADDR:-:2083}"
export STATIC_OVERRIDE_DIR="$PWD/data/dev/custom_code"

mkdir -p "$STATIC_OVERRIDE_DIR"

echo "=== VietPanel dev ==="
echo "config:       $VIETPANEL_CONFIG"
echo "secret:       $VIETPANEL_SECRET"
echo "translations: $VIETPANEL_TRANSLATIONS"
echo "listen:       $LISTEN_ADDR"
echo ""

exec go run ./cmd/panel

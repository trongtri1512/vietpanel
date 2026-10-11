#!/usr/bin/env bash
# VietPanel quick rebuild script - rebuilds binaries from latest source
# Usage: curl -fsSL https://raw.githubusercontent.com/trongtri1512/vietpanel/main/build.sh | bash
set -euo pipefail

log(){ echo -e "\n\033[1;34m[vietpanel-build]\033[0m $*"; }

[[ "$EUID" -ne 0 ]] && { echo "run as root"; exit 1; }

log "checking go toolchain"
if ! command -v go >/dev/null || ! go version | grep -qE 'go1\.(2[5-9]|[3-9][0-9])'; then
  log "installing go 1.25"
  ARCH=$(dpkg --print-architecture)
  [[ "$ARCH" == "amd64" ]] && GOARCH=amd64 || GOARCH=arm64
  curl -fsSL "https://go.dev/dl/go1.25.0.linux-${GOARCH}.tar.gz" -o /tmp/go.tgz
  rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tgz
fi
export PATH=$PATH:/usr/local/go/bin

log "pulling latest code"
cd /opt/vietpanel
git pull origin main 2>/dev/null || true

log "building binaries"
mkdir -p /usr/local/vietpanel
(cd panel && go build -o /usr/local/vietpanel/panel ./cmd/panel && go build -o /usr/local/vietpanel/genhash ./cmd/genhash)
(cd admin && go build -o /usr/local/vietpanel/admin ./cmd/admin)
(cd license-server && go build -o /usr/local/vietpanel/license-server .)

log "restarting services"
systemctl restart vietpanel-panel vietpanel-admin vietpanel-license

log "done!"
echo ""
echo "VietPanel rebuilt successfully."
echo "  user panel : https://$(grep -oP '(?<=hostname=).*' /etc/openpanel/openpanel/conf/openpanel.config 2>/dev/null || echo 'your-domain'):2083"
echo "  admin      : https://$(grep -oP '(?<=hostname=).*' /etc/openpanel/openpanel/conf/openpanel.config 2>/dev/null || echo 'your-domain'):2087"

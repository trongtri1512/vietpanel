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
if [[ ! -d /opt/vietpanel/.git ]]; then
  # A non-git copy here would silently rebuild from stale source, so refuse
  # to reuse it and clone fresh instead.
  if [[ -d /opt/vietpanel ]]; then
    stale="/opt/vietpanel.stale.$(date +%Y%m%d%H%M)"
    log "/opt/vietpanel is not a git checkout - moving it to $stale and cloning fresh"
    mv /opt/vietpanel "$stale"
  fi
  git clone --depth 1 https://github.com/trongtri1512/vietpanel.git /opt/vietpanel
fi
cd /opt/vietpanel
git fetch origin main && git reset --hard origin/main
log "building at $(git log --oneline -1)"

log "building binaries"
mkdir -p /usr/local/vietpanel
# Build to .new then swap, so an interrupted build can't leave a truncated
# binary for systemd to exec.
build(){ (cd "$1" && go build -o "/usr/local/vietpanel/$2.new" $3) && mv "/usr/local/vietpanel/$2.new" "/usr/local/vietpanel/$2"; }
build panel panel ./cmd/panel
build panel genhash ./cmd/genhash
build admin admin ./cmd/admin
build license-server license-server .

log "restarting services"
# vietpanel-license is optional and absent on systemd-only installs.
units=()
for u in vietpanel-panel vietpanel-admin vietpanel-license; do
  systemctl list-unit-files "$u.service" --no-legend | grep -q . && units+=("$u")
done
[[ ${#units[@]} -gt 0 ]] && systemctl restart "${units[@]}"

log "done!"
echo ""
echo "VietPanel rebuilt successfully."
echo "  user panel : https://$(grep -oP '(?<=hostname=).*' /etc/openpanel/openpanel/conf/openpanel.config 2>/dev/null || echo 'your-domain'):2083"
echo "  admin      : https://$(grep -oP '(?<=hostname=).*' /etc/openpanel/openpanel/conf/openpanel.config 2>/dev/null || echo 'your-domain'):2087"

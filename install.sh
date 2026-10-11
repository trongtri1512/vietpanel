#!/usr/bin/env bash
# VietPanel one-shot installer for Ubuntu 22.04+/Debian 12 (root required).
#
#   curl -fsSL https://raw.githubusercontent.com/trongtri1512/vietpanel/main/install.sh | bash -s -- \
#        --domain vietpanel.trongtri.com --email you@example.com
#
# Installs: MySQL, Redis, Go toolchain, rootless podman runtime (podman +
# podman-compose + shared image store, same model as upstream OpenPanel),
# VietPanel user panel (:2083),
# VietAdmin (:2087), self-hosted license-server (:2095), CLI (opencli/vietcli),
# and the front end from the root stack: the openpanel/caddy-coraza container on
# :80/:443 (serves every customer domain, with per-domain OWASP WAF) and the
# bind9 container authoritative on :53. TLS for --domain comes from Let's Encrypt
# via Caddy; the panel and admin terminate their own TLS on 2083/2087.
set -euo pipefail

DOMAIN=""; EMAIL=""; REPO="https://github.com/trongtri1512/vietpanel.git"; BRANCH="main"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain) DOMAIN="$2"; shift 2;;
    --email) EMAIL="$2"; shift 2;;
    --repo) REPO="$2"; shift 2;;
    --branch) BRANCH="$2"; shift 2;;
    *) echo "unknown arg: $1"; exit 1;;
  esac
done
[[ -z "$DOMAIN" ]] && { echo "--domain is required"; exit 1; }
[[ "$EUID" -ne 0 ]] && { echo "run as root"; exit 1; }
EMAIL="${EMAIL:-admin@$DOMAIN}"

log(){ echo -e "\n\033[1;34m[vietpanel]\033[0m $*"; }

log "0/9 preflight"
. /etc/os-release
case "$ID" in ubuntu|debian) ;; *) echo "unsupported OS: $ID (use Ubuntu 22.04+/Debian 12)"; exit 1;; esac
PUBIP=$(curl -fsS --max-time 8 https://ifconfig.me || echo "")
RESOLVED=$(getent hosts "$DOMAIN" | awk '{print $1; exit}' || true)
if [[ -n "$PUBIP" && -n "$RESOLVED" && "$RESOLVED" != "$PUBIP" ]]; then
  log "WARNING: $DOMAIN resolves to $RESOLVED but this server is $PUBIP - TLS will fail until DNS is fixed"
fi
# ifconfig.me can be unreachable; fall back to the first global address like
# upstream's setup_bind() does with `hostname -I`.
SERVER_IP="${PUBIP:-$(hostname -I | awk '{print $1}')}"
[[ -n "$SERVER_IP" ]] || { echo "FATAL: could not determine this server's IP address"; exit 1; }

log "1/9 packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
# No host nginx and no host caddy: the front end is the containerised
# openpanel/caddy-coraza from the root stack (same as upstream PODMAN_INSTALL.sh,
# which installs neither). The distro caddy also cannot parse the coraza_waf{}
# blocks in caddy/templates/domain.conf.
apt-get install -y -qq git curl ca-certificates ufw redis-server mysql-server sqlite3 \
  php-fpm php-mysql php-curl php-gd php-mbstring php-xml php-zip php-intl php-bcmath \
  php-imagick php-opcache php-cli >/dev/null
# Ubuntu 22.04 gives php8.1-fpm, Ubuntu 24.04 gives php8.3-fpm. Enable whatever
# apt actually installed so user websites survive a server reboot.
PHP_FPM_UNIT=$(systemctl list-unit-files --type=service --no-legend 'php*-fpm' 2>/dev/null | awk '{print $1; exit}')
if [ -n "$PHP_FPM_UNIT" ]; then
  systemctl enable --now "$PHP_FPM_UNIT"
fi
# Both used to be the front end here. Mask rather than just disable, so an
# upgrade over an older install cannot leave one of them holding :80/:443
# and Caddy would then fail to bind.
for svc in nginx caddy; do
  systemctl disable --now "$svc" >/dev/null 2>&1 || true
  systemctl mask "$svc" >/dev/null 2>&1 || true
done

# mysql-server only ships the `mysql` name; ~150 vietcli call sites invoke `mariadb`.
# Upstream always creates this shim (PODMAN_INSTALL.sh:652) and install.sh used to skip it,
# which made every DB-backed verb die with exit 127.
if ! command -v mariadb >/dev/null 2>&1; then
  if command -v mysql >/dev/null 2>&1; then
    ln -sf "$(command -v mysql)" /usr/local/bin/mariadb
  fi
fi

# container runtime package set, mirrors upstream PODMAN_INSTALL.sh install_packages()
if [[ -f /etc/needrestart/needrestart.conf ]]; then
  sed -i "s/#\$nrconf{restart} = 'i';/\$nrconf{restart} = 'a';/g" /etc/needrestart/needrestart.conf
fi
echo 'APT::Acquire::Retries "3";' > /etc/apt/apt.conf.d/80-retries
apt-get install -y -qq apt-transport-https ca-certificates >/dev/null
update-ca-certificates >/dev/null 2>&1 || true
KERNEL_PKG="linux-image-amd64"
if [[ "$ID" == "ubuntu" ]]; then KERNEL_PKG="linux-generic"; fi
apt-get install -y -qq openssl cron tar dbus-user-session systemd dbus uidmap iptables \
  podman default-mysql-client >/dev/null
# upstream always installs a kernel metapackage, for vendor kernels that predate user
# namespaces. On Ubuntu that drags in ~1GB of desktop/GPU/wireless firmware and replaces
# the default boot entry, so skip it when a distro-supported flavour is already running.
if ! uname -r | grep -qE -- '-(generic|cloud|virtual|lowlatency|aws|azure|gcp|kvm)'; then
  apt-get install -y -qq "$KERNEL_PKG" >/dev/null 2>&1 \
    || apt-get install -y -qq linux-image >/dev/null 2>&1 \
    || echo "[vietpanel] warning: kernel metapackage ($KERNEL_PKG) install failed"
fi
# upstream treats these as warn-not-die: older releases may not package them
apt-get install -y -qq systemd-container quota crun netavark aardvark-dns slirp4netns passt \
  fuse-overlayfs >/dev/null \
  || echo "[vietpanel] warning: some podman helper packages unavailable; podman may fall back to CNI/runc"
# Debian 12 / Ubuntu 22.04 ship no podman-compose package: pip venv fallback (upstream :353-359)
if ! apt-get install -y -qq podman-compose >/dev/null 2>&1 || ! command -v podman-compose >/dev/null 2>&1; then
  apt-get install -y -qq python3-venv >/dev/null
  if [[ ! -x /opt/podman-compose/bin/podman-compose ]]; then
    python3 -m venv /opt/podman-compose
    /opt/podman-compose/bin/pip install -q -U pip podman-compose
    ln -sf /opt/podman-compose/bin/podman-compose /usr/local/bin/podman-compose
  fi
  hash -r
fi
if ! command -v podman >/dev/null 2>&1 || ! command -v podman-compose >/dev/null 2>&1; then
  echo "FATAL: podman runtime packages missing after install"; exit 1
fi

log "1b/9 podman runtime"
SHARED_STORE=/var/lib/containers/shared-storage

# rootless podman is impossible with empty /etc/sub{uid,gid}; keep 100000-165535 free
# because cli/user/add.sh:307-308 hands exactly that range to new accounts via usermod.
touch /etc/subuid /etc/subgid
grep -q '^root:' /etc/subuid || echo 'root:231072:65536' >> /etc/subuid
grep -q '^root:' /etc/subgid || echo 'root:231072:65536' >> /etc/subgid

# upstream runs `podman system reset -f` unconditionally (golden-image stale storage),
# but that wipes storage - only reset when there are no images/containers to lose.
if [[ -z "$(podman images -q 2>/dev/null || true)" && -z "$(podman ps -aq 2>/dev/null || true)" ]]; then
  podman system reset -f >/dev/null 2>&1 || true
  rm -rf /root/.local/share/containers/libpod
  find /var/lib/containers/storage /run/containers/storage -mindepth 1 -delete 2>/dev/null || true
fi
install -d -Z /var/lib/containers 2>/dev/null || install -d /var/lib/containers

mkdir -p /etc/containers/registries.conf.d
echo 'unqualified-search-registries = ["docker.io"]' > /etc/containers/registries.conf.d/99-openpanel.conf

# pasta is missing on some hosts; upstream guards this block the same way
if command -v pasta >/dev/null 2>&1; then
  mkdir -p /etc/containers/containers.conf.d
  printf '[network]\ndefault_rootless_network_cmd = "pasta"\n' > /etc/containers/containers.conf.d/99-openpanel-net.conf
fi

mkdir -p /var/lib/containers/storage /run/containers/storage "$SHARED_STORE"
# shared store must stay world-readable or rootless per-user podmans cannot use additionalimagestores
chmod -R o+rX "$SHARED_STORE"
find "$SHARED_STORE" -name '*.lock' -exec chmod o+rw {} \;

cat > /etc/containers/storage.conf <<EOF
[storage]
driver = "overlay"
graphroot = "/var/lib/containers/storage"
runroot = "/run/containers/storage"

[storage.options]
additionalimagestores = [
  "$SHARED_STORE"
]

[storage.options.overlay]
ignore_chown_errors = "true"
EOF

# SELinux relabel; no-op on Ubuntu, kept for upstream parity
if command -v getenforce >/dev/null 2>&1 && [[ "$(getenforce 2>/dev/null || true)" != "Disabled" ]] && command -v restorecon >/dev/null 2>&1; then
  restorecon -RF /var/lib/containers/storage /run/containers/storage "$SHARED_STORE" >/dev/null 2>&1 || true
fi

if ! podman_info_err=$(podman info 2>&1 >/dev/null); then
  if [[ "$podman_info_err" == *"graph driver"*"does not match"* || "$podman_info_err" == *"database configuration mismatch"* ]]; then
    echo "[vietpanel] stale podman storage state - resetting and retrying"
    podman system reset -f >/dev/null 2>&1 || true
    rm -rf /var/lib/containers/storage/* /run/containers/storage/* 2>/dev/null || true
    podman info >/dev/null 2>&1 || { echo "FATAL: podman failed to initialize"; exit 1; }
  else
    echo "FATAL: podman failed to initialize: $(tail -3 <<< "$podman_info_err")"; exit 1
  fi
fi

# upstream enables podman.socket only AFTER storage.conf exists - keep that ordering
systemctl enable --now podman.socket

# cli/lib/podman.sh addresses each account's socket as /hostfs/run/user/<uid>/podman/podman.sock
# so one path works whether vietcli runs on the host or inside the panel container. Without this
# symlink every per-account podman call fails and user-add dies with "Podman not responding".
ln -sf / /hostfs

echo "[vietpanel] pulling shared images into $SHARED_STORE"
podman --root "$SHARED_STORE" pull docker.io/library/alpine:latest >/dev/null \
  || { echo "FATAL: failed to pull alpine into shared store"; exit 1; }
podman --root "$SHARED_STORE" pull docker.io/ubuntu/bind9:latest >/dev/null 2>&1 \
  || echo "[vietpanel] warning: bind9 image pull failed"
chmod -R o+rX "$SHARED_STORE"
find "$SHARED_STORE" -name '*.lock' -exec chmod o+rw {} \; 2>/dev/null || true
if command -v restorecon >/dev/null 2>&1; then restorecon -R "$SHARED_STORE" >/dev/null 2>&1 || true; fi

log "2/9 go toolchain"
if ! command -v go >/dev/null || ! go version | grep -qE 'go1\.(2[5-9]|[3-9][0-9])'; then
  ARCH=$(dpkg --print-architecture); [[ "$ARCH" == "amd64" ]] && GOARCH=amd64 || GOARCH=arm64
  curl -fsSL "https://go.dev/dl/go1.25.0.linux-${GOARCH}.tar.gz" -o /tmp/go.tgz
  rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tgz
fi
export PATH=$PATH:/usr/local/go/bin

log "3/9 source"
# reuse a pre-staged source tree (private repos, air-gapped installs); clone only when absent
if [[ -f /opt/vietpanel/install.sh && -d /opt/vietpanel/panel ]]; then
  log "using pre-staged source at /opt/vietpanel"
else
  rm -rf /opt/vietpanel
  git clone --depth 1 --branch "$BRANCH" "$REPO" /opt/vietpanel
fi
cd /opt/vietpanel

log "4/9 build"
mkdir -p /usr/local/vietpanel
(cd panel && go build -o /usr/local/vietpanel/panel ./cmd/panel && go build -o /usr/local/vietpanel/genhash ./cmd/genhash)
(cd admin && go build -o /usr/local/vietpanel/admin ./cmd/admin)
(cd license-server && go build -o /usr/local/vietpanel/license-server .)

log "5/9 mysql"
systemctl enable --now mysql
DBPASS=$(head -c 18 /dev/urandom | base64 | tr -d '/+=' | cut -c1-24)
mysql <<SQL
CREATE DATABASE IF NOT EXISTS panel;
CREATE USER IF NOT EXISTS 'vietpanel'@'localhost' IDENTIFIED BY '$DBPASS';
GRANT ALL ON panel.* TO 'vietpanel'@'localhost';
FLUSH PRIVILEGES;
SQL
mysql panel < configuration/mysql/initialize/1.1/plans.sql
cat > /etc/my.cnf <<EOF
[client]
user=vietpanel
password=$DBPASS
host=127.0.0.1
port=3306
database=panel
EOF
chmod 600 /etc/my.cnf

log "6/9 redis (unix socket)"
# Ubuntu's unit has PrivateTmp=true + restricted ReadWritePaths, so a socket under
# /tmp needs a drop-in and the dir must be writable by the redis user.
mkdir -p /tmp/redis && chown redis:redis /tmp/redis && chmod 775 /tmp/redis
# Ensure /tmp/redis is recreated after reboot (tmpfs clears /tmp)
echo 'd /tmp/redis 0775 redis redis -' > /etc/tmpfiles.d/redis.conf
systemd-tmpfiles --create 2>/dev/null || true
mkdir -p /etc/systemd/system/redis-server.service.d
cat > /etc/systemd/system/redis-server.service.d/override.conf <<'EOF'
[Service]
PrivateTmp=false
ReadWritePaths=/tmp/redis /var/lib/redis /var/log/redis /var/run/redis
EOF
grep -q '^unixsocket /tmp/redis/redis.sock' /etc/redis/redis.conf 2>/dev/null || cat >> /etc/redis/redis.conf <<'EOF'
unixsocket /tmp/redis/redis.sock
unixsocketperm 777
EOF
systemctl daemon-reload
systemctl reset-failed redis-server 2>/dev/null || true
systemctl restart redis-server
sleep 1
systemctl is-active --quiet redis-server || { echo "FATAL: redis did not start"; journalctl -u redis-server -n 20 --no-pager; exit 1; }

log "7/9 panel config + translations + license-server"
mkdir -p /etc/openpanel/openpanel/conf /etc/openpanel/openadmin/config /var/lib/vietpanel
# Ship the whole configuration tree to /etc/openpanel/ like upstream download_files() does.
# The CLI reads its templates from here:
#   nginx vhost tpl    cli/domains/add.sh:493  /etc/openpanel/nginx/vhosts/1.1/docker_<ws>_domain.conf
#   caddy domain tpl   cli/domains/add.sh:547  /etc/openpanel/caddy/templates/domain.conf
#   bind9 zone tpl     cli/domains/add.sh:613  /etc/openpanel/bind9/zone_template.txt
#   per-user compose   cli/user/add.sh:494,501 /etc/openpanel/docker/compose/1.0/{docker-compose.yml,.env}
#   per-user php.ini   cli/user/add.sh:582     /etc/openpanel/php/ini/
#   per-user service cfgs (mysql/postgres/nginx/openresty/ols/apache/varnish/ofelia/
#   backups) and skeleton: cli/user/add.sh:573-581,596
cp -a configuration/. /etc/openpanel/
mkdir -p /etc/openpanel/caddy/domains  # cli/domains/add.sh:546 writes <domain>.conf here
# NOTE: caddy/templates/domain.conf uses coraza_waf directives that only the
# coraza-patched caddy image from docker-compose parses; stock /usr/bin/caddy cannot.

# Root stack. cli/lib/podman.sh's podman_compose_file() resolves the ""/default/root
# context to /root/docker-compose.yml, and podman_ensure_running does
# `cd /root && podman-compose up -d <service>` for caddy and openpanel_dns.
# Only those two are ever started: openpanel/openpanel_mysql/openpanel_redis are
# served here by the vietpanel-panel + vietpanel-admin units and host mysql/redis,
# and their published ports (2083, 127.0.0.1:3306, /tmp/redis/redis.sock) would
# collide with them.
cp configuration/docker/compose/docker-compose.yml /root/docker-compose.yml
cp configuration/docker/compose/.env /root/.env
# upstream setup_bind(): pin bind9's published :53 to the server's real IP, because
# podman's aardvark-dns already holds :53 on every bridge gateway address.
sed -i -E 's#^( *- *")([0-9.]+:)?(53:53/(tcp|udp)")#\1'"$SERVER_IP"':\3#' /root/docker-compose.yml

# Caddy host-mounts /var/log/caddy for per-domain access logs, coraza audit logs
# and goaccess stats; the paths come from cli/user/{transfer,restore}.sh.
mkdir -p /var/log/caddy/stats /var/log/caddy/domlogs /var/log/caddy/coraza_waf
# Must be a real directory, not a symlink into /var/lib/caddy: the container's
# /data/caddy/certificates is bind-mounted from here, and
# admin/internal/bootstrap/caddyfile.go reads the issued certs back out of the same
# path to give the panel/admin TLS on 2083/2087. Migrate a pre-existing symlink
# (older installers pointed it at the distro caddy's storage) and keep the cert.
ACME_DIR=/etc/openpanel/caddy/ssl/acme-v02.api.letsencrypt.org-directory
if [[ -L "$ACME_DIR" ]]; then
  acme_target=$(readlink -f "$ACME_DIR")
  rm "$ACME_DIR"
  mkdir -p "$ACME_DIR"
  [[ -d "$acme_target" ]] && cp -a "$acme_target/." "$ACME_DIR/" || true
fi
mkdir -p "$ACME_DIR" /etc/openpanel/caddy/ssl/certs /etc/openpanel/caddy/ssl/custom

# live named config: the bind9 container mounts /etc/bind and cli/domains/* manage zones there
install -d -m 755 /etc/bind
# named.conf.local holds the live `zone ...` lines that cli/domains/add.sh appends,
# so refreshing it from the template on a re-run would silently unserve every domain.
if [[ -f /etc/bind/named.conf.local ]]; then
  find configuration/bind9/. -maxdepth 1 -mindepth 1 ! -name named.conf.local \
    -exec cp -a {} /etc/bind/ \;
else
  cp -a configuration/bind9/. /etc/bind/
fi
mkdir -p /etc/bind/zones
cp configuration/openpanel/conf/openpanel.config /etc/openpanel/openpanel/conf/openpanel.config
cp -r translations /etc/openpanel/openpanel/translations
# features/*.txt drive the sidebar + dashboard icons; core/service/websites/custom_code
# are runtime assets. Without them the panel renders with an empty nav/dashboard.
for d in features core service websites custom_code; do
  [ -d "configuration/openpanel/$d" ] && { rm -rf "/etc/openpanel/openpanel/$d"; cp -r "configuration/openpanel/$d" /etc/openpanel/openpanel/; }
done
head -c 32 /dev/urandom | base64 > /etc/openpanel/openpanel/secret.key
chmod 600 /etc/openpanel/openpanel/secret.key
sed -i 's/^dev_mode=.*/dev_mode=off/' /etc/openpanel/openpanel/conf/openpanel.config

ADMIN_TOKEN=$(head -c 24 /dev/urandom | base64 | tr -d '/+=' )
cat > /etc/systemd/system/vietpanel-license.service <<EOF
[Unit]
Description=VietPanel license server
After=network.target
[Service]
ExecStart=/usr/local/vietpanel/license-server
Environment=LICENSE_DB=/var/lib/vietpanel/licenses.json
Environment=LICENSE_ADDR=127.0.0.1:2095
Environment=LICENSE_ADMIN_TOKEN=$ADMIN_TOKEN
Restart=always
[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload && systemctl enable --now vietpanel-license
sleep 1
LICENSE_KEY=$(curl -fsS -X POST http://127.0.0.1:2095/api/keys \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H 'Content-Type: application/json' \
  -d '{"plan":"enterprise","note":"self-hosted installer"}' | python3 -c 'import sys,json;print(json.load(sys.stdin)["key"])')
python3 - "$LICENSE_KEY" <<'PY'
import sys, re
p = "/etc/openpanel/openpanel/conf/openpanel.config"
s = open(p).read()
key = sys.argv[1]
s = re.sub(r"(?m)^key=.*$", f"key={key}", s, count=1)
if "[LICENSE]" in s and "license_server_url=" not in s:
    s = s.replace("[LICENSE]", "[LICENSE]\nlicense_server_url=http://127.0.0.1:2095/enterprise/index.php", 1)
open(p, "w").write(s)
print("license key installed:", key)
PY

log "8/9 services + cli + admin"
# Copy the entire openadmin/config dir - admin panel reads admin.ini,
# features.json (drives /settings/modules), notifications.ini, services.json,
# shortcuts.json, log_paths.json, terms.md, etc. from here.
cp -r configuration/openadmin/config/. /etc/openpanel/openadmin/config/
head -c 32 /dev/urandom | base64 > /etc/openpanel/openadmin/secret.key
chmod 600 /etc/openpanel/openadmin/secret.key

install -d /usr/local/vietcli
cp -r cli/. /usr/local/vietcli/
chmod +x /usr/local/vietcli/vietcli 2>/dev/null || true
ln -sf /usr/local/vietcli/vietcli /usr/local/bin/vietcli
ln -sf /usr/local/vietcli/vietcli /usr/local/bin/opencli

for svc in panel admin; do
cat > /etc/systemd/system/vietpanel-$svc.service <<EOF
[Unit]
Description=VietPanel $svc
After=network.target mysql.service redis-server.service
[Service]
ExecStart=/usr/local/vietpanel/$svc
Restart=always
[Install]
WantedBy=multi-user.target
EOF
done
systemctl daemon-reload
systemctl enable --now vietpanel-panel vietpanel-admin

# seed a demo hosting account so you can log in immediately
DEMO_PASS=$(head -c 9 /dev/urandom | base64 | tr -d '/+=' | cut -c1-12)
HASH=$(/usr/local/vietpanel/genhash "$DEMO_PASS")
mysql panel <<SQL
INSERT INTO users (id, username, password, email, user_domains, twofa_enabled, plan, server)
VALUES (1, 'demo', '$HASH', 'demo@$DOMAIN', 0, 0, 'default', 'default')
ON DUPLICATE KEY UPDATE password='$HASH';
SQL

# first super admin for the admin panel (:2087) - vietcli needs sqlite3
ADMIN_PASS=$(head -c 9 /dev/urandom | base64 | tr -d '/+=' | cut -c1-14)
vietcli admin new admin "$ADMIN_PASS" --super >/dev/null 2>&1 || true
cat > /root/vp-credentials.txt <<CREDS
PANEL_LOGIN demo / $DEMO_PASS
ADMIN_LOGIN admin / $ADMIN_PASS
CREDS

log "9/9 front end: caddy-coraza + bind9 + firewall"
# Full upstream Caddyfile layout. The general block turns on on_demand_tls, whose
# `ask` endpoint (check.conf) only answers 200 for a domain that has a
# /etc/openpanel/caddy/domains/<name>.conf -- that is what stops a stranger who
# points their own domain at this IP from making Caddy mint certs for it. The
# trailing imports are what actually load the per-domain confs that
# `vietcli domains-add` writes; without them customer sites are configured but
# never served.
#
# The HOSTNAME block differs from upstream's `reverse_proxy localhost:2087`
# because our admin terminates its own TLS on 2087. Its markers must survive:
# build.sh, cli/{admin,domain,faq,sentinel}.sh, cli/lib/email.sh,
# cli/user/{add,login}.sh and admin/internal/bootstrap/caddyfile.go all parse the
# domain and the admin port back out of it.
cat > /etc/openpanel/caddy/Caddyfile <<EOF
# general
{
    #admin off

    on_demand_tls {
        ask      "http://localhost/check"
    }
}

# START HOSTNAME DOMAIN #
$DOMAIN {
    reverse_proxy https://localhost:2087 {
        transport http {
            tls_insecure_skip_verify
        }
    }
}
# END HOSTNAME DOMAIN #

# START WEBMAIL DOMAIN #
# no webmail container here; 'vietcli email-webmail' rewrites this block
# END WEBMAIL DOMAIN #

# START PHPMYADMIN DOMAIN #
# no phpmyadmin container here; 'vietcli phpmyadmin set <domain>' rewrites this block
# END PHPMYADMIN DOMAIN #


# import /check route
import /etc/openpanel/caddy/check.conf

# import domain_log
import /etc/openpanel/caddy/templates/domain_log

# import wp-manager rules
import /etc/openpanel/caddy/templates/wp.rules

# import cloudflare only IP addresses
import /etc/openpanel/caddy/templates/cloudflare.only

# import all sites
import /etc/openpanel/caddy/domains/*
EOF

# upstream configure_caddy_extras(): the /openpanel*, /openadmin* and /webmail*
# shortcuts injected into every customer site still carry the example.net placeholder.
sed -i "s/example\.net/$SERVER_IP/g" /etc/openpanel/caddy/redirects.conf
# the on_demand_tls ask URL is http://localhost/check, answered by Caddy itself
grep -qE '^127\.0\.0\.1[[:space:]]+localhost' /etc/hosts || echo "127.0.0.1 localhost" >> /etc/hosts

ufw allow 22,2083,2087/tcp >/dev/null 2>&1 || true
ufw allow 80,443/tcp >/dev/null 2>&1 || true  # caddy (network_mode: host) + ACME HTTP-01
# bind9 authoritative only: named.conf.options sets `recursion yes` with
# `allow-recursion { localhost; 192.168.1.0/24; 172.0.0.0/8; }`, so outside queries
# for a zone we do not host come back REFUSED rather than turning us into an open resolver.
ufw allow 53 >/dev/null 2>&1 || true
# UFW sets the FORWARD policy to DROP and netavark only ACCEPTs RELATED,ESTABLISHED
# inbound, so ports published by the root stack's bridge are unreachable from
# outside without this route rule. Per-account containers are unaffected: rootless
# podman forwards through a userspace proxy bound to 127.0.0.1, never via FORWARD.
EXT_IF=$(ip route get 1.1.1.1 2>/dev/null | grep -oP 'dev \K\S+' | head -1)
if [[ -n "$EXT_IF" ]]; then
  ufw route allow in on "$EXT_IF" to 172.20.0.0/24 >/dev/null 2>&1 || true
fi
ufw --force enable >/dev/null 2>&1 || true

# named.conf.local includes /etc/bind/rndc.key, so bind9 refuses to start without it.
if [[ ! -f /etc/bind/rndc.key ]]; then
  podman pull -q docker.io/ubuntu/bind9:latest >/dev/null 2>&1 || true
  timeout 120 podman run --rm -v /etc/bind/:/etc/bind/:Z --entrypoint=/bin/sh \
    docker.io/ubuntu/bind9:latest \
    -c 'rndc-confgen -a -A hmac-sha256 -b 256 -c /etc/bind/rndc.key' >/dev/null 2>&1 \
    || log "WARNING: could not generate /etc/bind/rndc.key - bind9 will not start until it exists"
fi
find /etc/bind/ -type d -print0 | xargs -0 chmod 755
find /etc/bind/ -type f -print0 | xargs -0 chmod 644

# upstream configure_waf(): pulls coraza_rules.conf + OWASP CRS + the app-profile
# exclusion plugins, flips SecRuleEngine On in every domain conf, points
# /root/.env at openpanel/caddy-coraza and ends with `podman-compose up -d caddy`.
# The CRS has to be on disk first -- caddy/templates/domain.conf Includes it by
# absolute path, so Caddy will not even parse its config without it.
log "fetching OWASP CRS and starting the front end (this clones a few repos)"
vietcli waf enable </dev/null >/tmp/vp-waf-enable.log 2>&1 \
  || log "WARNING: 'vietcli waf enable' failed, see /tmp/vp-waf-enable.log"
(cd /root && podman-compose up -d caddy bind9 </dev/null >/tmp/vp-compose-up.log 2>&1) \
  || log "WARNING: could not start caddy/bind9, see /tmp/vp-compose-up.log"

# upstream generate_ssl(): let Caddy finish ACME for the hostname before restarting
# the panel/admin, which LoadX509KeyPair() once at boot.
for _attempt in 1 2 3; do
  curl -4 -sf -o /dev/null --max-time 20 "https://$DOMAIN" && break
  podman restart caddy >/dev/null 2>&1 || true
  sleep 5
done
systemctl restart vietpanel-admin vietpanel-panel

if podman ps --format '{{.Names}}' | grep -qx caddy; then
  log "front end up: caddy on :80/:443 (WAF $(grep -rlE 'SecRuleEngine (On|DetectionOnly)' /etc/openpanel/caddy/domains/*.conf 2>/dev/null | wc -l) domain(s) protected)"
else
  log "WARNING: the caddy container is not running - customer domains will not be served."
  log "         podman logs caddy"
fi
if podman ps --format '{{.Names}}' | grep -qx openpanel_dns; then
  log "authoritative DNS up on $SERVER_IP:53"
else
  log "WARNING: openpanel_dns is not running - podman logs openpanel_dns"
fi

cat <<EOF

============================================================
 VietPanel installed.
   user panel : https://$DOMAIN:2083
   admin      : https://$DOMAIN:2087
   panel login: demo / $DEMO_PASS
   admin login: admin / $ADMIN_PASS
   (saved to /root/vp-credentials.txt)
   license key: $LICENSE_KEY  (validated by local license-server :2095)
   license admin token (save it): $ADMIN_TOKEN
     issue more keys:
       curl -X POST http://127.0.0.1:2095/api/keys -H "Authorization: Bearer $ADMIN_TOKEN" -H 'Content-Type: application/json' -d '{"plan":"enterprise"}'
   front end  : caddy-coraza container on :80/:443 (per-domain WAF),
                bind9 container authoritative on $SERVER_IP:53
   DNS: point an A record for $DOMAIN to $SERVER_IP (used for the 2083/2087 certs).
   Customer sites: 'vietcli domains-add <domain> <user>', then point that
   domain's A record at $SERVER_IP -- Caddy issues its cert on first request.
============================================================
EOF

#!/usr/bin/env bash
# VietPanel one-shot installer for Ubuntu 22.04+/Debian 12 (root required).
#
#   curl -fsSL https://raw.githubusercontent.com/trongtri1512/vietpanel/main/install.sh | bash -s -- \
#        --domain vietpanel.trongtri.com --email you@example.com
#
# Installs: MySQL, Redis, Caddy, Go toolchain, VietPanel user panel (:2083),
# VietAdmin (:2087), self-hosted license-server (:2095), CLI (opencli/vietcli),
# and wires TLS for --domain (and admin.--domain) via Caddy/Let's Encrypt.
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

log "1/9 packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq git curl ca-certificates ufw redis-server caddy mysql-server sqlite3 \
  php-fpm php-mysql php-curl php-gd php-mbstring php-xml php-zip php-intl php-bcmath \
  php-imagick php-opcache php-cli nginx >/dev/null
# Ubuntu 22.04 gives php8.1-fpm, Ubuntu 24.04 gives php8.3-fpm. Enable whatever
# apt actually installed so user websites survive a server reboot.
PHP_FPM_UNIT=$(systemctl list-unit-files --type=service --no-legend 'php*-fpm' 2>/dev/null | awk '{print $1; exit}')
if [ -n "$PHP_FPM_UNIT" ]; then
  systemctl enable --now "$PHP_FPM_UNIT"
fi
systemctl enable --now nginx

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

log "9/9 caddy + firewall"
# openpanel-style block: the panel/admin read the domain from this block and each
# serves its own TLS on 2083/2087, so Caddy proxies to the admin over https.
CADDY_CONF="# START HOSTNAME DOMAIN #
$DOMAIN {
    reverse_proxy https://localhost:2087 {
        transport http {
            tls_insecure_skip_verify
        }
    }
}
# END HOSTNAME DOMAIN #"
mkdir -p /etc/openpanel/caddy/ssl/custom
ln -sfn /var/lib/caddy/.local/share/caddy/certificates/acme-v02.api.letsencrypt.org-directory /etc/openpanel/caddy/ssl/acme-v02.api.letsencrypt.org-directory
printf '%s\n' "$CADDY_CONF" > /etc/openpanel/caddy/Caddyfile   # read by panel/admin/cli
printf '%s\n' "$CADDY_CONF" > /etc/caddy/Caddyfile             # the file Ubuntu's caddy.service actually loads
# 80/443 stay closed: the panel is reached on 2083 and the admin on 2087.
# Caddy is disabled; the Caddyfile is kept only because the panel/admin parse the
# domain (and admin port) out of the HOSTNAME DOMAIN block.
systemctl disable --now caddy >/dev/null 2>&1 || true

# If caddy previously issued LE certs for $DOMAIN, they live under /var/lib/caddy
# with 600:caddy:caddy perms - admin/panel run as root but still need the parent
# dir traversable. Open the perms so both binaries can LoadX509KeyPair().
if [[ -d "/etc/openpanel/caddy/ssl/acme-v02.api.letsencrypt.org-directory/$DOMAIN" ]]; then
  chmod -R a+rX /etc/openpanel/caddy/ssl/acme-v02.api.letsencrypt.org-directory/ 2>/dev/null || true
  chmod 644 "/etc/openpanel/caddy/ssl/acme-v02.api.letsencrypt.org-directory/$DOMAIN/$DOMAIN.key" 2>/dev/null || true
  systemctl restart vietpanel-admin vietpanel-panel
else
  log "no LE cert for $DOMAIN yet - panel/admin will serve plain HTTP."
  log "to issue one: enable caddy, open port 80, run 'caddy run --config /etc/caddy/Caddyfile', then re-run this installer."
fi
ufw allow 22,2083,2087/tcp >/dev/null 2>&1 || true
ufw --force enable >/dev/null 2>&1 || true

cat <<EOF

============================================================
 VietPanel installed.
   user panel : https://$DOMAIN:2083
   admin      : https://$DOMAIN:2087
   panel login: demo / $DEMO_PASS
   admin login: admin / $ADMIN_PASS
   (saved to /root/vp-credentials.txt; ports 80/443 are closed by design)
   license key: $LICENSE_KEY  (validated by local license-server :2095)
   license admin token (save it): $ADMIN_TOKEN
     issue more keys:
       curl -X POST http://127.0.0.1:2095/api/keys -H "Authorization: Bearer $ADMIN_TOKEN" -H 'Content-Type: application/json' -d '{"plan":"enterprise"}'
   DNS: point an A record for $DOMAIN to $PUBIP (used for the 2083/2087 certs).
============================================================
EOF

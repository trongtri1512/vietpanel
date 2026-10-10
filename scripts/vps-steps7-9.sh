#!/usr/bin/env bash
# VietPanel steps 7-9 (resume after redis fix). Idempotent-ish, run as root.
set -euo pipefail
cd /opt/vietpanel
DOMAIN="vietpanel.trongtri.com"
LICENSE_URL="https://license.vietpanel.trongtri.com/enterprise/index.php"
LICENSE_KEY="enterprise-6915cacf2b23a82d2f606d8dac8f4518"

echo "[7/9] panel config + translations"
mkdir -p /etc/openpanel/openpanel/conf /etc/openpanel/openadmin/config
cp configuration/openpanel/conf/openpanel.config /etc/openpanel/openpanel/conf/openpanel.config
rm -rf /etc/openpanel/openpanel/translations
cp -r translations /etc/openpanel/openpanel/translations
head -c 32 /dev/urandom | base64 > /etc/openpanel/openpanel/secret.key
chmod 600 /etc/openpanel/openpanel/secret.key
sed -i 's/^dev_mode=.*/dev_mode=off/' /etc/openpanel/openpanel/conf/openpanel.config
python3 - "$LICENSE_URL" "$LICENSE_KEY" <<'PY'
import re, sys
url, key = sys.argv[1], sys.argv[2]
p = "/etc/openpanel/openpanel/conf/openpanel.config"
s = open(p).read()
s = re.sub(r"(?m)^key=.*$", f"key={key}", s, count=1)
if "license_server_url=" in s:
    s = re.sub(r"(?m)^license_server_url=.*$", f"license_server_url={url}", s)
else:
    s = s.replace("[LICENSE]", f"[LICENSE]\nlicense_server_url={url}", 1)
open(p, "w").write(s)
print("config: license_server_url + key set")
PY

echo "[8/9] admin + cli + services"
cp configuration/openadmin/config/admin.ini /etc/openpanel/openadmin/config/admin.ini
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

DEMO_PASS=$(head -c 9 /dev/urandom | base64 | tr -d '/+=' | cut -c1-12)
HASH=$(/usr/local/vietpanel/genhash "$DEMO_PASS")
mysql panel <<SQL
INSERT INTO users (id, username, password, email, user_domains, twofa_enabled, plan, server)
VALUES (1, 'demo', '$HASH', 'demo@$DOMAIN', 0, 0, 'default', 'default')
ON DUPLICATE KEY UPDATE password='$HASH';
SQL
echo "DEMO_LOGIN demo / $DEMO_PASS" > /root/vp-credentials.txt

echo "[9/9] caddy + firewall"
cat > /etc/caddy/Caddyfile <<EOF
$DOMAIN {
    reverse_proxy 127.0.0.1:2083
}
admin.$DOMAIN {
    reverse_proxy 127.0.0.1:2087
}
EOF
systemctl enable --now caddy
ufw allow 22,80,443,2083,2087/tcp >/dev/null 2>&1 || true
ufw --force enable >/dev/null 2>&1 || true

echo "DONE"
cat /root/vp-credentials.txt
#!/bin/bash
# One-shot dev setup: start Homebrew MySQL, create panel schema + test user,
# write the MySQL option file, and restart the panel against it.
set -e
cd "$(dirname "$0")/.."

DB_USER="${DB_USER:-root}"
DB_PASS="${DB_PASS:-}"          # Homebrew mysql root has empty password by default
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-3306}"
SCHEMA="${SCHEMA:-/Users/trongtri/Desktop/Project/vietpanel/configuration/mysql/initialize/1.1/plans.sql}"

echo "== start mysql =="
brew services start mysql >/dev/null 2>&1 || mysql.server start >/dev/null 2>&1 || true
for i in $(seq 1 30); do
  if mysqladmin ping --silent 2>/dev/null; then echo "mysql up"; break; fi
  sleep 2
done

echo "== create schema =="
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" ${DB_PASS:+-p"$DB_PASS"} -e "CREATE DATABASE IF NOT EXISTS panel;"
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" ${DB_PASS:+-p"$DB_PASS"} panel < "$SCHEMA"

echo "== seed test user (demo / demo123) =="
HASH=$(go run ./cmd/genhash demo123)
cat > /tmp/vietpanel_seed.sql <<SQL
INSERT INTO users (id, username, password, email, user_domains, twofa_enabled, plan, server)
VALUES (1, 'demo', '$HASH', 'demo@example.com', 0, 0, 'default', 'default')
ON DUPLICATE KEY UPDATE password='$HASH';
SQL
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" ${DB_PASS:+-p"$DB_PASS"} panel < /tmp/vietpanel_seed.sql

echo "== write my.cnf =="
cat > data/dev/my.cnf <<EOF
[client]
user=$DB_USER
password=$DB_PASS
host=$DB_HOST
port=$DB_PORT
database=panel
EOF

echo "== restart panel =="
pkill -f "bin/panel" 2>/dev/null || true
sleep 1
export VIETPANEL_CONFIG="$PWD/data/dev/conf/openpanel.config"
export VIETPANEL_SECRET="$PWD/data/dev/secret.key"
export VIETPANEL_TRANSLATIONS="/Users/trongtri/Desktop/Project/vietpanel/translations/"
export VIETPANEL_MYSQL_OPTFILE="$PWD/data/dev/my.cnf"
export LISTEN_ADDR=":2083"
export STATIC_OVERRIDE_DIR="$PWD/data/dev/custom_code"
mkdir -p "$STATIC_OVERRIDE_DIR"
nohup ./bin/panel > data/dev/panel.log 2>&1 &
sleep 4
grep -E "MySQL pool|listening" data/dev/panel.log | tail -3
echo "== done: login at http://localhost:2083/login (demo / demo123) =="

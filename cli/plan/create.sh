#!/usr/bin/env bash
# Description: Create a new hosting plan.
# Usage: vietcli plan-create <name> [--domains <n>] [--websites <n>] [--disk <val>] [--cpu <n>] [--ram <val>]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"
NAME="" DOMAINS="0" WEBSITES="0" DISK="0" CPU="1" RAM="1G" DB_LIMIT="0"
while [[ $# -gt 0 ]]; do case "$1" in
    --domains) DOMAINS="$2"; shift 2 ;; --websites) WEBSITES="$2"; shift 2 ;;
    --disk) DISK="$2"; shift 2 ;; --cpu) CPU="$2"; shift 2 ;; --ram) RAM="$2"; shift 2 ;; --db-limit) DB_LIMIT="$2"; shift 2 ;;
    --*) die "Unknown option: $1" ;; *) NAME="$1"; shift ;;
esac; done
[[ -n "$NAME" ]] || die "Usage: vietcli plan-create <name> [options]"
db_available || die "Database not available."
EXISTS=$(db_query "SELECT COUNT(*) FROM plans WHERE name = '$(db_escape "$NAME")'")
[[ "$EXISTS" -eq 0 ]] || die "Plan '$NAME' already exists."
db_query "INSERT INTO plans (name, domains_limit, websites_limit, disk_limit, cpu, ram, db_limit) VALUES ('$(db_escape "$NAME")', '$DOMAINS', '$WEBSITES', '$DISK', '$CPU', '$RAM', '$DB_LIMIT')" || die "Failed to create plan."
log "Plan '$NAME' created." && echo "Plan '$NAME' created." && echo "  Domains: $DOMAINS  Websites: $WEBSITES  Disk: $DISK  CPU: $CPU  RAM: $RAM"

#!/usr/bin/env bash
# Description: Delete a domain.
# Usage: vietcli domain-delete <domain>

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"
DOMAIN="${1:-}"; [[ -n "$DOMAIN" ]] || die "Usage: vietcli domain-delete <domain>"
DOMAIN=$(echo "$DOMAIN" | tr '[:upper:]' '[:lower:]')
db_available || die "Database not available."
EXISTS=$(db_query "SELECT COUNT(*) FROM domains WHERE domain_url = '$(db_escape "$DOMAIN")'")
[[ "$EXISTS" -gt 0 ]] || die "Domain '$DOMAIN' not found."
db_query "DELETE FROM sites WHERE domain_id = (SELECT domain_id FROM domains WHERE domain_url = '$(db_escape "$DOMAIN")')" || true
db_query "DELETE FROM domains WHERE domain_url = '$(db_escape "$DOMAIN")'" || die "Failed to delete domain."
log "Domain '$DOMAIN' deleted." && echo "Domain '$DOMAIN' deleted successfully."

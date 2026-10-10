#!/usr/bin/env bash
# Description: List domains.
# Usage: vietcli domain-list [<username>]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"
db_available || die "Database not available."
USERNAME="${1:-}"
printf "%-8s %-35s %-15s %-30s\n" "ID" "DOMAIN" "USER" "DOCROOT"
printf "%.0s-" {1..90}; echo
if [[ -n "$USERNAME" ]]; then
    db_query "SELECT d.domain_id, d.domain_url, u.username, d.docroot FROM domains d JOIN users u ON u.id = d.user_id WHERE u.username = '$(db_escape "$USERNAME")' ORDER BY d.domain_id" | while IFS=$'\t' read -r id domain user docroot; do printf "%-8s %-35s %-15s %-30s\n" "$id" "$domain" "$user" "$docroot"; done
else
    db_query "SELECT d.domain_id, d.domain_url, u.username, d.docroot FROM domains d JOIN users u ON u.id = d.user_id ORDER BY d.domain_id" | while IFS=$'\t' read -r id domain user docroot; do printf "%-8s %-35s %-15s %-30s\n" "$id" "$domain" "$user" "$docroot"; done
fi

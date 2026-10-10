#!/usr/bin/env bash
# Description: Add a domain to a user.
# Usage: vietcli domain-add <domain> <username> [--docroot <path>]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"
DOMAIN="" USERNAME="" DOCROOT="/var/www/html"
while [[ $# -gt 0 ]]; do case "$1" in
    --docroot) DOCROOT="$2"; shift 2 ;; --*) die "Unknown option: $1" ;;
    *) [[ -z "$DOMAIN" ]] && { DOMAIN="$1"; shift; continue; }
       [[ -z "$USERNAME" ]] && { USERNAME="$1"; shift; continue; }
       die "Too many arguments." ;;
esac; done
[[ -n "$DOMAIN" && -n "$USERNAME" ]] || die "Usage: vietcli domain-add <domain> <username> [--docroot <path>]"
db_available || die "Database not available."
DOMAIN=$(echo "$DOMAIN" | tr '[:upper:]' '[:lower:]' | sed 's/\.$//')
[[ "$DOMAIN" =~ ^[a-zA-Z0-9.-]+$ ]] || die "Invalid domain."
USER_ID=$(db_query "SELECT id FROM users WHERE username = '$(db_escape "$USERNAME")'" | head -1)
[[ -n "$USER_ID" ]] || die "User '$USERNAME' not found."
EXISTS=$(db_query "SELECT COUNT(*) FROM domains WHERE domain_url = '$(db_escape "$DOMAIN")'")
[[ "$EXISTS" -eq 0 ]] || die "Domain '$DOMAIN' is already registered."
db_query "INSERT INTO domains (domain_url, user_id, docroot, php_version) VALUES ('$(db_escape "$DOMAIN")', $USER_ID, '$(db_escape "$DOCROOT")', '8.2')" || die "Failed to add domain."
log "Domain '$DOMAIN' added for '$USERNAME'." && echo "Domain '$DOMAIN' added."

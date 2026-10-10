#!/usr/bin/env bash
# Description: Unsuspend a hosting user.
# Usage: vietcli user-unsuspend <username>

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"

USERNAME="${1:-}"
[[ -n "$USERNAME" ]] || die "Usage: vietcli user-unsuspend <username>"
db_available || die "Database not available."
EXISTS=$(db_query "SELECT COUNT(*) FROM users WHERE username = '$(db_escape "$USERNAME")'")
[[ "$EXISTS" -gt 0 ]] || die "User '$USERNAME' not found."
db_query "UPDATE users SET server = 'active' WHERE username = '$(db_escape "$USERNAME")'" || die "Failed to unsuspend user."
log "User '$USERNAME' unsuspended." && echo "User '$USERNAME' unsuspended."

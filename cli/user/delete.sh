#!/usr/bin/env bash
# Description: Delete a hosting user.
# Usage: vietcli user-delete <username>

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"

USERNAME="${1:-}"
[[ -n "$USERNAME" ]] || die "Usage: vietcli user-delete <username>"
db_available || die "Database not available."
EXISTS=$(db_query "SELECT COUNT(*) FROM users WHERE username = '$(db_escape "$USERNAME")'")
[[ "$EXISTS" -gt 0 ]] || die "User '$USERNAME' not found."
db_query "DELETE FROM users WHERE username = '$(db_escape "$USERNAME")'" || die "Failed to delete user."
log "User '$USERNAME' deleted."
echo "User '$USERNAME' deleted successfully."

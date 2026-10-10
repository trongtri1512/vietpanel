#!/usr/bin/env bash
# Description: Suspend a hosting user.
# Usage: vietcli user-suspend <username> [--comment <reason>]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"

USERNAME="" COMMENT=""
while [[ $# -gt 0 ]]; do case "$1" in
    --comment) COMMENT="$2"; shift 2 ;;
    *) USERNAME="$1"; shift ;;
esac; done
[[ -n "$USERNAME" ]] || die "Usage: vietcli user-suspend <username> [--comment <reason>]"
db_available || die "Database not available."
EXISTS=$(db_query "SELECT COUNT(*) FROM users WHERE username = '$(db_escape "$USERNAME")'")
[[ "$EXISTS" -gt 0 ]] || die "User '$USERNAME' not found."
SUSPENDED_VAL="suspended"; [[ -n "$COMMENT" ]] && SUSPENDED_VAL="suspended: $COMMENT"
db_query "UPDATE users SET server = '$(db_escape "$SUSPENDED_VAL")' WHERE username = '$(db_escape "$USERNAME")'" || die "Failed to suspend user."
log "User '$USERNAME' suspended." && echo "User '$USERNAME' suspended."

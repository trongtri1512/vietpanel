#!/usr/bin/env bash
# Description: Change a user's password.
# Usage: vietcli user-password <username> [--password <new_pw>]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"

USERNAME="" PASSWORD=""
while [[ $# -gt 0 ]]; do case "$1" in
    --password) PASSWORD="$2"; shift 2 ;;
    *) USERNAME="$1"; shift ;;
esac; done
[[ -n "$USERNAME" ]] || die "Usage: vietcli user-password <username> [--password <new_pw>]"
db_available || die "Database not available."
EXISTS=$(db_query "SELECT COUNT(*) FROM users WHERE username = '$(db_escape "$USERNAME")'")
[[ "$EXISTS" -gt 0 ]] || die "User '$USERNAME' not found."
if [[ -z "$PASSWORD" || "$PASSWORD" == "generate" ]]; then PASSWORD=$(generate_password 16); log "Generated new password: $PASSWORD"; fi
HASH=""
if command -v python3 &>/dev/null; then HASH=$(python3 -c "import bcrypt; print(bcrypt.hashpw(b'$PASSWORD', bcrypt.gensalt()).decode())" 2>/dev/null); fi
[[ -n "$HASH" ]] || die "Cannot hash password."
db_query "UPDATE users SET password = '$(db_escape "$HASH")' WHERE username = '$(db_escape "$USERNAME")'" || die "Failed to update password."
log "Password changed for '$USERNAME'." && echo "Password updated for '$USERNAME'." && echo "New password: $PASSWORD"

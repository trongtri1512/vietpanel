#!/usr/bin/env bash
# Description: Add a new hosting user.
# Usage: vietcli user-add <username> <plan_name> [--password <pw>] [--email <email>]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"

USERNAME="" PLAN_NAME="" PASSWORD="" EMAIL=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --password) PASSWORD="$2"; shift 2 ;;
        --email)    EMAIL="$2"; shift 2 ;;
        --*)        die "Unknown option: $1" ;;
        *) [[ -z "$USERNAME" ]] && { USERNAME="$1"; shift; continue; }
           [[ -z "$PLAN_NAME" ]] && { PLAN_NAME="$1"; shift; continue; }
           die "Too many arguments." ;;
    esac
done

[[ -n "$USERNAME" && -n "$PLAN_NAME" ]] || die "Usage: vietcli user-add <username> <plan_name> [--password <pw>] [--email <email>]"
db_available || die "Database not available."
(( ${#USERNAME} >= 3 && ${#USERNAME} <= 20 )) || die "Username must be 3–20 characters."
[[ "$USERNAME" =~ ^[a-zA-Z][a-zA-Z0-9]*$ ]] || die "Username must start with a letter and contain only letters and numbers."

PLAN_ID=$(db_query "SELECT id FROM plans WHERE name = '$(db_escape "$PLAN_NAME")'" | head -1)
[[ -n "$PLAN_ID" ]] || die "Plan '$PLAN_NAME' not found."
EXISTING=$(db_query "SELECT COUNT(*) FROM users WHERE username = '$(db_escape "$USERNAME")'")
[[ "$EXISTING" -eq 0 ]] || die "Username '$USERNAME' is already taken."

if [[ -z "$PASSWORD" || "$PASSWORD" == "generate" ]]; then
    PASSWORD=$(generate_password 16)
    log "Generated password: $PASSWORD"
fi

HASH=""
if command -v python3 &>/dev/null; then
    HASH=$(python3 -c "import bcrypt; print(bcrypt.hashpw(b'$PASSWORD', bcrypt.gensalt()).decode())" 2>/dev/null)
fi
[[ -n "$HASH" ]] || die "Cannot hash password. Install bcrypt Python module."
[[ -n "$EMAIL" ]] || EMAIL="${USERNAME}@vietpanel.local"

REG_DATE=$(date '+%Y-%m-%d %H:%M:%S')
case "$DB_BACKEND" in
    sqlite) sqlite3 "$VIETCLI_DB_SQLITE" "INSERT INTO users (username, password, email, plan, plan_id, registered_date, server) VALUES ('$(db_escape "$USERNAME")', '$(db_escape "$HASH")', '$(db_escape "$EMAIL")', '$(db_escape "$PLAN_NAME")', $PLAN_ID, '$REG_DATE', 'active');" 2>/dev/null || die "Failed to insert user." ;;
    mysql)  mariadb --defaults-extra-file="$config_file" -D "$mysql_database" -sN -e "INSERT INTO users (username, password, email, plan, plan_id, registered_date, server) VALUES ('$(db_escape "$USERNAME")', '$(db_escape "$HASH")', '$(db_escape "$EMAIL")', '$(db_escape "$PLAN_NAME")', $PLAN_ID, '$REG_DATE', 'active');" ;;
esac

log "User '$USERNAME' added successfully (plan: $PLAN_NAME)."
echo "Username: $USERNAME" && echo "Password: $PASSWORD" && echo "Email:    $EMAIL" && echo "Plan:     $PLAN_NAME"

#!/usr/bin/env bash
# ======================================================================
# Shared database helper sourced by every vietcli command that needs
# to query the panel database.
#
# Production: reads MySQL via /etc/my.cnf (database: panel).
# Development: falls back to SQLite when VIETCLI_DB_SQLITE is set
#              (e.g. VIETCLI_DB_SQLITE="./data/panel.db").
#
# All scripts that source this file get:
#   db_query "<sql>"       — runs a query and prints the result
#   db_escape "<value>"    — escapes a value for safe SQL use
# ======================================================================

config_file="/etc/my.cnf"
mysql_database="panel"

# --- SQLite dev mode ---
if [[ -n "${VIETCLI_DB_SQLITE:-}" && -f "$VIETCLI_DB_SQLITE" ]]; then
    DB_BACKEND="sqlite"
else
    DB_BACKEND="mysql"
fi

# escapes backslashes and single quotes so a value is safe inside a
# single-quoted SQL literal.
db_escape() {
    local s="$1"
    s="${s//\\/\\\\}"
    s="${s//\'/\\\'}"
    printf '%s' "$s"
}

# prints the result of a query to stdout. On MySQL this uses
# mariadb --defaults-extra-file; on SQLite it uses sqlite3.
db_query() {
    local sql="$1"
    case "$DB_BACKEND" in
        sqlite)
            sqlite3 -separator $'\t' "$VIETCLI_DB_SQLITE" "$sql" 2>/dev/null
            ;;
        mysql)
            [[ -f "$config_file" ]] || {
                echo "FATAL ERROR: MariaDB config file not available: $config_file"
                exit 1
            }
            mariadb --defaults-extra-file="$config_file" -D "$mysql_database" -sN -e "$sql" 2>/dev/null
            ;;
    esac
}

# passes when DB is available (MySQL or SQLite)
db_available() {
    case "$DB_BACKEND" in
        sqlite) [[ -f "$VIETCLI_DB_SQLITE" ]] ;;
        mysql)  [[ -f "$config_file" ]] && mariadb --defaults-extra-file="$config_file" -e "SELECT 1" &>/dev/null ;;
    esac
}
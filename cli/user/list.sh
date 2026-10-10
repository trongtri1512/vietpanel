#!/usr/bin/env bash
# Description: List all panel users with their plan and status.
# Usage: vietcli user-list

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"

db_available || die "Database not available."

printf "%-5s %-20s %-15s %-10s %-20s\n" "ID" "USERNAME" "PLAN" "STATUS" "REGISTERED"
printf "%.0s-" {1..75}; echo

db_query "SELECT id, username, COALESCE(plan,'-'), COALESCE(server,'active'), COALESCE(registered_date,'-')
FROM users ORDER BY id" | while IFS=$'\t' read -r id username plan status registered; do
    printf "%-5s %-20s %-15s %-10s %-20s\n" "$id" "$username" "$plan" "$status" "$registered"
done
#!/usr/bin/env bash
# Description: List all hosting plans.
# Usage: vietcli plan-list

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CLI_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
source "$CLI_DIR/lib/db.sh"
source "$CLI_DIR/lib/helpers.sh"
db_available || die "Database not available."
printf "%-5s %-20s %-10s %-10s %-8s %-8s %-8s %-8s\n" "ID" "NAME" "DOMAINS" "WEBSITES" "DB_LIM" "DISK" "CPU" "RAM"
printf "%.0s-" {1..85}; echo
db_query "SELECT id, name, COALESCE(domains_limit,'-'), COALESCE(websites_limit,'-'), COALESCE(db_limit,'-'), COALESCE(disk_limit,'-'), COALESCE(cpu,'-'), COALESCE(ram,'-') FROM plans ORDER BY id" | while IFS=$'\t' read -r id name domains websites db disk cpu ram; do
    printf "%-5s %-20s %-10s %-10s %-8s %-8s %-8s %-8s\n" "$id" "$name" "$domains" "$websites" "$db" "$disk" "$cpu" "$ram"
done

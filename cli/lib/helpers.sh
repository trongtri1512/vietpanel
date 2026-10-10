#!/usr/bin/env bash
# ======================================================================
# Shared helpers sourced by vietcli command scripts.
#   log "<message>"          — prints timestamped message to stderr
#   die "<message>"          — logs and exits with code 1
#   is_linux                 — true on Linux
#   is_macos                 — true on macOS (Darwin)
#   linux_only               — dies on non-Linux
#   require_command <cmd>    — installs via pkg manager if missing
#   generate_password [len]  — prints a random password (default 16)
# ======================================================================

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >&2
}

die() {
    echo "ERROR: $*" >&2
    exit 1
}

is_linux() {
    [[ "$(uname -s)" == "Linux" ]]
}

is_macos() {
    [[ "$(uname -s)" == "Darwin" ]]
}

linux_only() {
    is_linux || die "This command requires Linux. It cannot run on $(uname -s)."
}

require_command() {
    local cmd="$1"
    local package="${2:-$1}"

    command -v "$cmd" &>/dev/null && return 0

    if command -v apt-get &>/dev/null; then
        apt-get update -qq >/dev/null 2>&1
        apt-get install -y -qq "$package" >/dev/null 2>&1
    elif command -v dnf &>/dev/null; then
        dnf install -y -q "$package" >/dev/null 2>&1 || {
            dnf install -y -q epel-release >/dev/null 2>&1
            dnf install -y -q "$package" >/dev/null 2>&1
        }
    elif command -v yum &>/dev/null; then
        yum install -y -q "$package" >/dev/null 2>&1 || {
            yum install -y -q epel-release >/dev/null 2>&1
            yum install -y -q "$package" >/dev/null 2>&1
        }
    elif command -v brew &>/dev/null; then
        brew install "$package" >/dev/null 2>&1
    else
        echo "Error: No compatible package manager found. Please install $package manually."
        return 1
    fi

    command -v "$cmd" &>/dev/null || {
        echo "Error: $package installation failed. Please install $package manually."
        return 1
    }
}

generate_password() {
    local len="${1:-16}"
    LC_ALL=C tr -dc 'A-Za-z0-9!@#$%^&*()_+' </dev/urandom | head -c "$len"
}

# --- Enterprise check ---
# True when the panel config has key=enterprise (for gating Enterprise-only commands).
is_enterprise() {
    local config="${OPENPANEL_CONFIG:-/etc/openpanel/openpanel/conf/openpanel.config}"
    grep -qE '^key=enterprise' "$config" 2>/dev/null
}

ENTERPRISE_LINK="https://vietpanel.com/enterprise"
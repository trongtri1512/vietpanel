#!/bin/bash
################################################################################
# Script Name: license.sh
# Description: Manage VietPanel Enterprise license.
# Usage: vietcli license [key|verify|info|delete|<LICENSE_KEY>] [--json] [--no-restart]
# Author: Stefan Pejcic
# Created: 01.11.2023
# Last Modified: 21.08.2026
# Company: OpenPanel, LLC.
# Copyright (c) openpanel.com
# 
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
# 
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
# 
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.
################################################################################

# Configuration
readonly CONFIG_FILE_PATH='/etc/openpanel/openpanel/conf/openpanel.config'
# [LICENSE] license_server_url decides who validates the key, so a self-hosted
# license server works; upstream WHMCS is only the fallback when unset
license_endpoint="$(awk -F= '/^license_server_url[ \t]*=/{gsub(/[ \t\r]/,"",$2); print $2; exit}' "$CONFIG_FILE_PATH" 2>/dev/null)"
[[ -z "$license_endpoint" ]] && license_endpoint="https://api.openpanel.com/enterprise/index.php"
readonly WHMCS_URL="$license_endpoint"
readonly COMPOSE_FILE="/root/docker-compose.yml"
readonly SERVICE="openpanel"

readonly EMAIL_MODULES=(
    "emails"
    "webmail"
    "email_deliverability"
    "email_filters"
    "email_aliases"
    "email_import"
    "email_export"
    "email_default"
)

# Colors
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly RESET='\033[0m'

# Global flags
JSON="no"
NO_RESTART="no"

parse_flags() {
    JSON="no"
    NO_RESTART="no"

    for arg in "$@"; do
        case "$arg" in
            --json) JSON="yes" ;;
            --no-restart) NO_RESTART="yes" ;;
        esac
    done
}

usage() {
    cat << EOF
Usage: vietcli license [options]

Commands:
  key                                           View current license key.
  enterprise-XXXXXXXXXX                         Save an Enterprise license key.
  noc-XXXXXXXXXX                                Save a NOC license key.
  lifetime-XXXXXXXXXX                           Save a Lifetime license key.
  verify                                        Verify the license key.
  info                                          Display information about the license owner and expiration.
  delete                                        Delete the license key and downgrade VietPanel to Community edition.
EOF
    exit 1
}

read_config() {
    awk -F '=' '
        /\[LICENSE\]/{flag=1; next} 
        /\[/{flag=0} 
        flag{
            gsub(/^[ \t]+|[ \t]+$/, "", $1); 
            gsub(/^[ \t]+|[ \t]+$/, "", $2); 
            print $1 "=" $2
        }
    ' "$CONFIG_FILE_PATH"
}

get_license_key() {
    local config license_key
    config=$(read_config)
    license_key=$(echo "$config" | grep -i 'key' | cut -d'=' -f2)

    if [[ -z "$license_key" ]]; then
        if [[ "$JSON" == "yes" ]]; then
            echo "No License Key"
        else
            echo -e "${RED}No License Key${RESET}"
        fi
    else
        if [[ "$JSON" == "yes" ]]; then
            echo "$license_key"
        else
            echo -e "${GREEN}$license_key${RESET}"
        fi
    fi
}

get_public_ip() {
    curl --silent --max-time 1 -4 "https://ip.openpanel.com" || curl --silent --max-time 1 -4 "https://ifconfig.me/ip"
}

verify_license_api() {
    local license_key="$1"
    local ip_address check_token response
    
    ip_address=$(get_public_ip)
    check_token=$(openssl rand -hex 16)
    
    response=$(curl -sS --connect-timeout 3 --max-time 10 -X POST -d "licensekey=$license_key&ip=$ip_address&check_token=$check_token" "$WHMCS_URL")
    echo "$response" | grep -oP '(?<=<status>).*?(?=</status>)'
}

extract_xml_field() {
    local response="$1"
    local field="$2"
    echo "$response" | grep -oP "(?<=<$field>).*?(?=</$field>)"
}

output_message() {
    local message="$1"
    local color="${2:-}"

    if [[ "$JSON" == "yes" ]]; then
        echo "$message"
    else
        echo -e "${color}${message}${RESET}"
    fi
}

manage_compose_volumes() {
    local action="${1:-disable}"

    # a systemd install has no compose stack to mount extra volumes into
    [[ ! -f "$COMPOSE_FILE" ]] && return 0

    VOLUMES=(
    "/root/.ssh/:/root/.ssh/:ro"
    "/usr/local/mail/openmail/:/usr/local/mail/openmail/"
    "/etc/openpanel/openpanel/conf/knowledge_base_articles.json:/etc/openpanel/openpanel/conf/knowledge_base_articles.json:ro"
    )

    volume_exists() {
        local vol="$1"
        grep -Fq -- "- $vol" "$COMPOSE_FILE"
    }

    add_volumes() {
        START_LINE=$(awk "/$SERVICE:/ {flag=1} flag && /volumes:/ {print NR; exit}" "$COMPOSE_FILE")

        for vol in "${VOLUMES[@]}"; do
            if ! volume_exists "$vol"; then
                sed -i "$((START_LINE))a\      - $vol" "$COMPOSE_FILE"
                ((START_LINE++))
            fi
        done
    }

    remove_volumes() {
        for vol in "${VOLUMES[@]}"; do
            if volume_exists "$vol"; then
                sed -i "/^\s*-\s*${vol//\//\\/}\s*$/d" "$COMPOSE_FILE"
            fi
        done
    }

    if [ "$action" = "disable" ]; then
        remove_volumes
    else
        add_volumes
    fi
}

restart_services() {
    if [[ "$NO_RESTART" == "yes" ]]; then
        echo "Please restart VietAdmin service to enable new features."
        return 0
    fi

    # both services run on the host and this script is usually a child of VietAdmin itself,
    # so the restart is deferred to give the caller time to flush its HTTP response
    nohup bash -c 'sleep 2; systemctl restart vietpanel-panel vietpanel-admin 2>/dev/null || { service admin restart >/dev/null 2>&1; service openpanel restart >/dev/null 2>&1; }' &> /dev/null &
    disown

    echo "VietPanel and VietAdmin are restarted to apply Enterprise features."
}

toggle_email_modules() {
    local action="${1:-enable}"
    local enabled_modules new_modules module

    enabled_modules=$(grep '^enabled_modules=' "$CONFIG_FILE_PATH" | cut -d'=' -f2 | tr -d '"')
    new_modules="$enabled_modules"

    if [ "$action" = "disable" ]; then
        for module in "${EMAIL_MODULES[@]}"; do
            new_modules=$(echo "$new_modules" | sed "s/\b${module}\b//g; s/,,/,/g; s/^,//; s/,$//")
        done
    else
        for module in "${EMAIL_MODULES[@]}"; do
            if ! echo "$new_modules" | grep -q "\b${module}\b"; then
                if [ -z "$new_modules" ]; then
                    new_modules="$module"
                else
                    new_modules="${new_modules},${module}"
                fi
            fi
        done
    fi

    sed -i "s|^enabled_modules=.*|enabled_modules=\"${new_modules}\"|" "$CONFIG_FILE_PATH"
}

pagespeed_api_key_control() {
    local file key
    file="/etc/openpanel/openpanel/service/pagespeed.api"
    key="AIzaSyDow0GLE7N5gcZXa72tpqIvIaJtn1bDtsk"

    if [ "$1" = "remove" ]; then
        [ -f "$file" ] && grep -qx "$key" "$file" && rm -f "$file"
    else
        [ ! -s "$file" ] && echo "$key" > "$file"
    fi
}

save_license_to_file() {
    local new_key="$1"

    if vietcli config update key "$new_key" > /dev/null; then
        output_message "License key ${new_key} added." "$GREEN"
        toggle_email_modules "enable" > /dev/null
        vietcli config update how_to_guides no > /dev/null
        pagespeed_api_key_control > /dev/null
        manage_compose_volumes "enable"
        restart_services
    else
        output_message "License is valid, but failed to save the license key ${new_key}" "$RED"
    fi
}

verify_and_save_license() {
    local license_key="$1"
    local license_status
    license_status=$(verify_license_api "$license_key")

    if [[ "$license_status" == "Active" ]]; then
        save_license_to_file "$license_key"
    else
        output_message "License is invalid" "$RED"
        exit 1
    fi
}

verify_existing_license() {
    local config license_key license_status
    config=$(read_config)
    license_key=$(echo "$config" | grep -i 'key' | cut -d'=' -f2)

    if [[ -z "$license_key" ]]; then
        output_message "No License Key. Please add the key first: vietcli config update key XXXXXXXXXX" "$RED"
        exit 1
    fi

    license_status=$(verify_license_api "$license_key")
    if [[ "$license_status" == "Active" ]]; then
        output_message "License is valid" "$GREEN"
        manage_compose_volumes "enable"
        restart_services
    else
        output_message "License is invalid" "$RED"
        exit 1
    fi
}

show_license_info() {
    local config license_key response license_status ip_address check_token
    
    config=$(read_config)
    license_key=$(echo "$config" | grep -i 'key' | cut -d'=' -f2)

    if [[ -z "$license_key" ]]; then
        output_message "No License Key. Please add the key first." "$RED"
        exit 1
    fi

    ip_address=$(get_public_ip)
    check_token=$(openssl rand -hex 16)
    
    response=$(curl -sS --connect-timeout 3 --max-time 10 -X POST -d "licensekey=$license_key&ip=$ip_address&check_token=$check_token" "$WHMCS_URL")
    license_status=$(extract_xml_field "$response" "status")
    
    if [[ "$license_status" == "Active" ]]; then
        local registered_name company_name email product_name reg_date next_due_date billing_cycle valid_ip
        
        registered_name=$(extract_xml_field "$response" "registeredname")
        company_name=$(extract_xml_field "$response" "companyname")
        email=$(extract_xml_field "$response" "email")
        product_name=$(extract_xml_field "$response" "productname")
        reg_date=$(extract_xml_field "$response" "regdate")
        next_due_date=$(extract_xml_field "$response" "nextduedate")
        billing_cycle=$(extract_xml_field "$response" "billingcycle")
        valid_ip=$(extract_xml_field "$response" "validip")

        if [[ "$JSON" == "yes" ]]; then
            cat << EOF
{
    "Owner": "$registered_name",
    "Company Name": "$company_name",
    "Email": "$email",
    "License Type": "$product_name",
    "Registration Date": "$reg_date",
    "Next Due Date": "$next_due_date",
    "Billing Cycle": "$billing_cycle",
    "Valid IP": "$valid_ip"
}
EOF
        else
            cat << EOF
Owner: $registered_name
Company Name: $company_name
Email: $email
License Type: $product_name
Registration Date: $reg_date
Next Due Date: $next_due_date
Billing Cycle: $billing_cycle
Valid IP: $valid_ip
EOF
        fi
    else
        output_message "License is invalid" "$RED"
        exit 1
    fi
}

delete_license() {
    vietcli config update key "" > /dev/null
    toggle_email_modules "disable" > /dev/null
    pagespeed_api_key_control "remove" > /dev/null
    manage_compose_volumes
    restart_services
}

# Main
main() {
    local command="${1:-}"
    
    [ $# -eq 0 ] && usage

    parse_flags "$@"

    case "$command" in
        "key")                        license_key=$(get_license_key); echo "$license_key" ;;
        "info")                       show_license_info ;;
        "verify")                     verify_existing_license ;;
        "delete")                     delete_license ;;
        enterprise*|noc-*|lifetime-*) verify_and_save_license "$command" ;;
        *)                            output_message "Invalid command." "$RED"; usage ;;
    esac
}

main "$@"

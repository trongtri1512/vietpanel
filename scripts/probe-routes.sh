#!/usr/bin/env bash
# Probe every registered page route on both panels with an authenticated session.
P=https://vietpanel.trongtri.com:2083
A=https://vietpanel.trongtri.com:2087
JP=/tmp/jar_panel; JA=/tmp/jar_admin; rm -f "$JP" "$JA"

login() { # $1=base $2=jar $3=user $4=pass
  local t
  t=$(curl -sk -c "$2" "$1/login" | grep -oE 'name="csrf_token" value="[^"]+"' | head -1 | sed 's/.*value="//; s/"//')
  curl -sk -b "$2" -c "$2" -o /dev/null -X POST "$1/login" \
    --data-urlencode "username=$3" --data-urlencode "password=$4" --data-urlencode "csrf_token=$t"
  # sanity: dashboard reachable?
  curl -sk -b "$2" -o /dev/null -w "login(%3)=%{http_code}\n" "$1/dashboard"
}

echo "== PANEL (demo @ :2083) =="
login "$P" "$JP" demo I35dtNQ9CvZk
while read -r r; do
  [ -z "$r" ] && continue
  c=$(curl -sk -b "$JP" -o /dev/null -w '%{http_code}' "$P$r")
  printf '%s %s\n' "$c" "$r"
done < /root/panel_routes.txt | sort | uniq -c | sort -rn | head -5
while read -r r; do
  [ -z "$r" ] && continue
  c=$(curl -sk -b "$JP" -o /dev/null -w '%{http_code}' "$P$r")
  printf '%s %s\n' "$c" "$r"
done < /root/panel_routes.txt > /root/probe_panel.txt
echo "--- counts ---"; awk '{print $1}' /root/probe_panel.txt | sort | uniq -c | sort -rn
echo "--- non-2xx ---"; awk '$1!~/^2/' /root/probe_panel.txt

echo
echo "== ADMIN (admin @ :2087) =="
login "$A" "$JA" admin pjcjBcurGfJvRR
while read -r r; do
  [ -z "$r" ] && continue
  c=$(curl -sk -b "$JA" -o /dev/null -w '%{http_code}' "$A$r")
  printf '%s %s\n' "$c" "$r"
done < /root/admin_routes.txt > /root/probe_admin.txt
echo "--- counts ---"; awk '{print $1}' /root/probe_admin.txt | sort | uniq -c | sort -rn
echo "--- non-2xx (first 25) ---"; awk '$1!~/^2/' /root/probe_admin.txt | head -25
#!/bin/bash
#  nano /etc/profile.d/welcome.sh && chmod +x /etc/profile.d/welcome.sh
[ "$(id -u)" -ne 0 ] && return

# skip banner when launched from the VietAdmin > Advanced > Web Terminal
[ -n "$OPENPANEL_HIDE_WELCOME" ] && return 0 2>/dev/null || true

VERSION=$(/usr/local/bin/opencli version)
OPENADMIN_STATUS=$(/usr/local/bin/opencli admin)

echo -e  "================================================================"
echo -e  ""
echo -e "  🚀 VietPanel ${VERSION}"
echo -e  ""
echo -e  "$OPENADMIN_STATUS"
echo -e  ""
echo -e "  📖 Resources:"
echo -e "     • Docs:     https://github.com/trongtri1512/vietpanel/admin/intro/"
echo -e "     • Forums:   https://community.openpanel.org/"
echo -e "     • Discord:  https://discord.openpanel.org/"
echo -e ""
echo -e "================================================================"

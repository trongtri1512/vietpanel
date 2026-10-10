#!/bin/bash
set -e

readonly PANEL="openpanel"
readonly GITHUB="stefanpejcic" # might be org eventually
readonly REPOSITORY="OpenPanel-Virtualizor"
readonly VIRTUALIZOR_PATH="/usr/local/virtualizor"

 
#1. check requirements
if [[ $EUID -ne 0 ]]; then
    echo "[✖] This script must be run as root!"
    exit 1
fi

if command -v virtualizor >/dev/null 2>&1; then
    echo "[ OK ] Virtualizor detected - proceeding with the install.."
else
    echo "[FAILED] Virtualizor binary not found! Is this a Virtualizor master server?"
    exit 1
fi


#2. set cleanup
cleanup() {
    echo
    echo "Cleaning up..."
    rm -rf "/tmp/$REPOSITORY"
}

trap cleanup EXIT INT TERM


#3. clone from github
echo
echo "Clonning $GITHUB/$REPOSITORY"
cd /tmp
git clone "https://github.com/$GITHUB/$REPOSITORY"


#4. https://youtu.be/tw429JGL5zo
echo
echo "Setup according to https://www.virtualizor.com/docs/admin/adding-custom-control-panel/"
SCRIPT="virt_${PANEL}.sh"
IMAGE="virt_${PANEL}.png"
JSON="${PANEL}_supported_os.json"

cd "$REPOSITORY"
cp "$SCRIPT" "$VIRTUALIZOR_PATH/hooks/$SCRIPT"
cp "$IMAGE"  "$VIRTUALIZOR_PATH/enduser/themes/default/images/$IMAGE"
cp "$JSON"   "$VIRTUALIZOR_PATH/$JSON"


#5. check
FILES=(
    "$VIRTUALIZOR_PATH/hooks/${SCRIPT}"
    "$VIRTUALIZOR_PATH/enduser/themes/default/images/${IMAGE}"
    "$VIRTUALIZOR_PATH/${JSON}"
)


missing=false
x_command="bash -x <(curl -sSL https://raw.githubusercontent.com/$GITHUB/$REPOSITORY/refs/heads/main/INSTALL.sh)"

echo
echo "Checking files.."
for f in "${FILES[@]}"; do
    if [[ -f "$f" ]]; then
        echo " [✔] $f"
    else
        echo " [✖] $f"
        missing=true
    fi
done
if [[ "$missing" = true ]]; then
    echo
    echo "[FAILED] Please re-run this command and share it's output on community.openpanel.org"
    echo
    echo "$x_command"
    exit 1
else
    echo
    echo "[OK] Installation completed successfully, $PANEL is now available on your Virtualizor server."
    exit 0
fi


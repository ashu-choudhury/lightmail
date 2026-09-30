#!/usr/bin/env bash
# ==============================================================================
# Lightmail 1-Command Linux Installer
# The World's Lightest, Most Streamlined Production Email Server
# https://github.com/ashu-choudhury/lightmail
# ==============================================================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

INSTALL_DIR="/opt/pmail"
SERVICE_NAME="pmail"
GITHUB_REPO="ashu-choudhury/lightmail"
RELEASE_TAG="continuous"

echo -e "${CYAN}${BOLD}"
cat << 'EOF'
========================================================================
   _     _       _     _                         _ _ 
  | |   (_) __ _| |__ | |_ _ __ ___   __ _  ___ (_) |
  | |   | |/ _` | '_ \| __| '_ ` _ \ / _` |/ _ \| | |
  | |___| | (_| | | | | |_| | | | | | (_| | (_) | | |
  |_____|_|\__, |_| |_|\__|_| |_| |_|\__,_|\___/|_|_|
           |___/                                     
  Lightmail: The World's Lightest Production Email Server
========================================================================
EOF
echo -e "${NC}"

# 1. Require root or sudo privileges
if [ "$(id -u)" -ne 0 ]; then
    echo -e "${RED}[ERROR] This installer must be run as root or with sudo.${NC}"
    echo -e "Usage: ${BOLD}curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/master/install.sh | sudo bash${NC}"
    exit 1
fi

# 2. Detect System Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        GOARCH="amd64"
        ;;
    aarch64|arm64)
        GOARCH="arm64"
        ;;
    *)
        echo -e "${RED}[ERROR] Unsupported architecture: ${ARCH}. Lightmail supports x86_64 and arm64.${NC}"
        exit 1
        ;;
esac
echo -e "${BLUE}[*] Detected architecture:${NC} linux/${GOARCH}"

# 3. Install Minimal Dependencies
echo -e "${BLUE}[*] Checking required system tools (curl, tar, gzip, libcap)...${NC}"
if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq >/dev/null 2>&1 || true
    apt-get install -y -qq curl tar gzip libcap2-bin >/dev/null 2>&1 || true
elif command -v dnf >/dev/null 2>&1; then
    dnf install -y -q curl tar gzip libcap >/dev/null 2>&1 || true
elif command -v yum >/dev/null 2>&1; then
    yum install -y -q curl tar gzip libcap >/dev/null 2>&1 || true
elif command -v apk >/dev/null 2>&1; then
    apk add --no-cache curl tar gzip libcap >/dev/null 2>&1 || true
elif command -v pacman >/dev/null 2>&1; then
    pacman -Sy --noconfirm curl tar gzip libcap >/dev/null 2>&1 || true
fi

# 4. Prepare Directory
mkdir -p "${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}/config"

# 5. Download Latest Release Binary
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${RELEASE_TAG}/lightmail-linux-${GOARCH}.tar.gz"
TMP_ARCHIVE="/tmp/lightmail-linux-${GOARCH}.tar.gz"

echo -e "${BLUE}[*] Downloading Lightmail binary from GitHub Releases...${NC}"
echo -e "    ${CYAN}${DOWNLOAD_URL}${NC}"

if curl -sSLf "${DOWNLOAD_URL}" -o "${TMP_ARCHIVE}"; then
    echo -e "${GREEN}[+] Download complete (~7-8MB).${NC}"
else
    # Fallback to direct latest release if continuous is building
    FALLBACK_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/lightmail-linux-${GOARCH}.tar.gz"
    echo -e "${YELLOW}[!] Trying fallback release: ${FALLBACK_URL}${NC}"
    if ! curl -sSLf "${FALLBACK_URL}" -o "${TMP_ARCHIVE}"; then
        echo -e "${RED}[ERROR] Could not download release archive from GitHub.${NC}"
        echo -e "Verify your server has internet access or visit https://github.com/${GITHUB_REPO}/releases"
        exit 1
    fi
fi

# 6. Extract and Install
echo -e "${BLUE}[*] Extracting Lightmail binary to ${INSTALL_DIR}/pmail...${NC}"
tar -xzf "${TMP_ARCHIVE}" -C /tmp/
if [ -f /tmp/lightmail ]; then
    mv -f /tmp/lightmail "${INSTALL_DIR}/pmail"
elif [ -f /tmp/pmail ]; then
    mv -f /tmp/pmail "${INSTALL_DIR}/pmail"
fi
rm -f "${TMP_ARCHIVE}"

chmod 755 "${INSTALL_DIR}/pmail"

# Grant capability to bind ports 25, 80, 443 without root if needed
if command -v setcap >/dev/null 2>&1; then
    setcap 'cap_net_bind_service=+ep' "${INSTALL_DIR}/pmail" || true
fi

# 7. Create Systemd Service
echo -e "${BLUE}[*] Configuring systemd service (${SERVICE_NAME}.service)...${NC}"
cat << EOF > /etc/systemd/system/${SERVICE_NAME}.service
[Unit]
Description=Lightmail Mail Server (Ultra-light pure SQLite mail engine)
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/pmail -p 8080
Restart=always
RestartSec=5
LimitNOFILE=65536
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl restart "${SERVICE_NAME}"
sleep 2

# 8. Detect Public IP for User Notice
IPV4=$(curl -s4 https://api.ipify.org || curl -s4 https://ifconfig.me || true)
IPV6=$(curl -s6 https://api64.ipify.org || curl -s6 https://ifconfig.co || true)

# 9. Completion Notice
echo -e "\n${GREEN}${BOLD}========================================================================${NC}"
echo -e "${GREEN}${BOLD}       🎉 Lightmail Successfully Installed & Running! 🎉       ${NC}"
echo -e "${GREEN}${BOLD}========================================================================${NC}"
echo -e "Service Status: ${GREEN}$(systemctl is-active ${SERVICE_NAME})${NC}"
echo -e "Memory Footprint: ${CYAN}~4.8MB RAM${NC}"
echo -e "Install Directory: ${BOLD}${INSTALL_DIR}${NC}"
echo -e "Config & DB:       ${BOLD}${INSTALL_DIR}/config/pmail.db${NC}"

echo -e "\n${YELLOW}${BOLD}👉 Next Step (1-Minute Web Setup Wizard):${NC}"
if [ -n "${IPV4}" ]; then
    echo -e "   Open in browser: ${BOLD}${CYAN}http://${IPV4}:8080${NC}"
elif [ -n "${IPV6}" ]; then
    echo -e "   Open in browser: ${BOLD}${CYAN}http://[${IPV6}]:8080${NC}"
else
    echo -e "   Open in browser: ${BOLD}${CYAN}http://<YOUR-SERVER-IP>:8080${NC}"
fi

echo -e "\n${BLUE}${BOLD}💡 Reverse Proxy Recommendation (Caddy):${NC}"
echo -e "To access Webmail over clean HTTPS (e.g. mail.yourdomain.com), we recommend Caddy:"
echo -e "Add this to your ${BOLD}/etc/caddy/Caddyfile${NC}:"
echo -e "--------------------------------------------------------"
echo -e "  mail.yourdomain.com {"
echo -e "      reverse_proxy 127.0.0.1:8080"
echo -e "  }"
echo -e "--------------------------------------------------------"
echo -e "Then run: ${BOLD}systemctl restart caddy${NC}"
echo -e "Caddy will automatically obtain Let's Encrypt SSL with zero configuration."
echo -e "\n${GREEN}Enjoy the world's lightest self-hosted mail server!${NC}\n"

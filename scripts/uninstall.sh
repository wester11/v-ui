#!/usr/bin/env bash
# void-wg uninstaller.
#   sudo bash uninstall.sh                # интерактивно
#   sudo VOIDWG_FORCE=1 bash uninstall.sh # без подтверждений (для CI)
set -Eeuo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/void-wg}"
FORCE="${VOIDWG_FORCE:-0}"

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'; NC='\033[0m'
log()  { printf "${GREEN}%s${NC}\n" "$*"; }
warn() { printf "${YELLOW}%s${NC}\n" "$*"; }
die()  { printf "${RED}%s${NC}\n" "$*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "Run as root"

confirm() {
    [ "$FORCE" = "1" ] && return 0
    local prompt="$1"; local ans
    read -r -p "$prompt [yes/N]: " ans
    [ "$ans" = "yes" ]
}

cat <<BANNER
${YELLOW}This will:
  - stop and remove void-wg containers
  - drop the postgres volume (ALL DATA WILL BE LOST)
  - remove the systemd unit
  - optionally remove ${INSTALL_DIR}${NC}
BANNER

if ! confirm "Proceed?"; then
    log "Aborted."
    exit 0
fi

if [ -d "$INSTALL_DIR" ]; then
    if [ -f "$INSTALL_DIR/docker-compose.yml" ]; then
        log "Stopping containers..."
        ( cd "$INSTALL_DIR" && docker compose --profile agent --profile metrics down -v --remove-orphans ) || true
    fi
fi

log "Removing v-wg CLI wrapper..."
rm -f /usr/local/bin/v-wg

log "Disabling systemd units (panel + TLS renewal timer)..."
systemctl stop void-wg.service 2>/dev/null || true
systemctl disable void-wg.service 2>/dev/null || true
systemctl stop void-wg-renew.timer 2>/dev/null || true
systemctl disable void-wg-renew.timer 2>/dev/null || true
rm -f /etc/systemd/system/void-wg.service \
      /etc/systemd/system/void-wg-renew.service \
      /etc/systemd/system/void-wg-renew.timer
systemctl daemon-reload

if confirm "Remove $INSTALL_DIR (this deletes your .env with all secrets)?"; then
    rm -rf "$INSTALL_DIR"
    log "Removed $INSTALL_DIR"
else
    warn "Kept $INSTALL_DIR (run again to remove)"
fi

# Закрыть открытые в ufw порты
if command -v ufw >/dev/null 2>&1; then
    log "Closing firewall ports..."
    for p in 51820/udp 51821/udp 80/tcp 443/tcp 8080/tcp; do
        ufw delete allow "$p" >/dev/null 2>&1 || true
    done
fi

if confirm "Remove Let's Encrypt certificates from /etc/letsencrypt?"; then
    rm -rf /etc/letsencrypt 2>/dev/null || true
    log "Removed /etc/letsencrypt"
fi

log "Uninstall complete."

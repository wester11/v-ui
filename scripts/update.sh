#!/usr/bin/env bash
# void-wg auto-updater. Идемпотентно: pull → rebuild → up.
set -Eeuo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/void-wg}"
REPO_BRANCH="${REPO_BRANCH:-main}"
LOG_FILE="${LOG_FILE:-/var/log/void-wg-update.log}"

GREEN='\033[0;32m'; RED='\033[0;31m'; YELLOW='\033[1;33m'; NC='\033[0m'
ts()   { date '+%Y-%m-%d %H:%M:%S'; }
log()  { printf "${GREEN}[%s] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
warn() { printf "${YELLOW}[%s] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
die()  { printf "${RED}[%s] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE" >&2; exit 1; }

trap 'die "Update failed at line $LINENO. See $LOG_FILE"' ERR

[ "$(id -u)" -eq 0 ]                || die "Run as root"
[ -d "$INSTALL_DIR/.git" ]          || die "Not installed: $INSTALL_DIR (.git missing)"
command -v docker >/dev/null 2>&1   || die "docker not found"
docker compose version >/dev/null 2>&1 || die "docker compose plugin missing"

mkdir -p "$(dirname "$LOG_FILE")"
: > "$LOG_FILE"
log "Update starting in $INSTALL_DIR (branch=$REPO_BRANCH)"

cd "$INSTALL_DIR"

log "Snapshotting .env -> .env.bak"
cp -f .env .env.bak 2>/dev/null || warn "no .env to back up"

log "Fetching latest..."
git fetch --quiet origin "$REPO_BRANCH"
LOCAL=$(git rev-parse HEAD)
REMOTE=$(git rev-parse "origin/$REPO_BRANCH")
if [ "$LOCAL" = "$REMOTE" ]; then
    log "Already up-to-date ($LOCAL)"
else
    log "Updating $LOCAL -> $REMOTE"
    git reset --hard "origin/$REPO_BRANCH"
fi

log "Pulling base images..."
docker compose pull --quiet 2>>"$LOG_FILE" || true

log "Rebuilding and restarting..."
docker compose up -d --build --remove-orphans >>"$LOG_FILE" 2>&1

log "Pruning dangling images..."
docker image prune -f >/dev/null

log "Waiting for API health..."
TLS_MODE="$(grep -E '^TLS_MODE=' .env 2>/dev/null | cut -d= -f2 | tr -d '"' || echo none)"
PANEL_HTTP_PORT="$(grep -E '^PANEL_HTTP_PORT=' .env 2>/dev/null | cut -d= -f2 | tr -d '"' || echo 80)"
PANEL_HTTPS_PORT="$(grep -E '^PANEL_HTTPS_PORT=' .env 2>/dev/null | cut -d= -f2 | tr -d '"' || echo 443)"
if [ "$TLS_MODE" = "none" ] || [ -z "$TLS_MODE" ]; then
    PROBE="http://127.0.0.1:${PANEL_HTTP_PORT}/healthz"
else
    PROBE="https://127.0.0.1:${PANEL_HTTPS_PORT}/healthz"
fi
for _ in $(seq 1 30); do
    if curl -fsSk "$PROBE" >/dev/null 2>&1; then
        log "API healthy ($PROBE) ✓"; break
    fi
    sleep 2
done

log "Update complete."

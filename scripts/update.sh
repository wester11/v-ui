#!/usr/bin/env bash
# Production-safe updater for void-wg.
# Idempotent, non-interactive, with rollback on failure.
set -Eeuo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/void-wg}"
REPO_BRANCH="${REPO_BRANCH:-main}"
LOG_FILE="${LOG_FILE:-/var/log/void-wg-install.log}"

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

ts()   { date '+%Y-%m-%d %H:%M:%S'; }
log()  { printf "${GREEN}[%s] [UPDATE] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
warn() { printf "${YELLOW}[%s] [WARN] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
err()  { printf "${RED}[%s] [ERROR] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE" >&2; }

OLD_HEAD=""
NEW_HEAD=""
ROLLBACK_DONE=0

require_root() {
  [ "$(id -u)" -eq 0 ] || { err "Run as root"; exit 1; }
}

require_env() {
  [ -d "$INSTALL_DIR/.git" ] || { err "Not installed: $INSTALL_DIR (.git missing)"; exit 1; }
  command -v git >/dev/null 2>&1 || { err "git not found"; exit 1; }
  command -v docker >/dev/null 2>&1 || { err "docker not found"; exit 1; }
  docker compose version >/dev/null 2>&1 || { err "docker compose plugin missing"; exit 1; }
}

probe_api_health() {
  local tls_mode http_port https_port probe
  tls_mode="$(grep -E '^TLS_MODE=' .env 2>/dev/null | cut -d= -f2 | tr -d '"' || true)"
  http_port="$(grep -E '^PANEL_HTTP_PORT=' .env 2>/dev/null | cut -d= -f2 | tr -d '"' || true)"
  https_port="$(grep -E '^PANEL_HTTPS_PORT=' .env 2>/dev/null | cut -d= -f2 | tr -d '"' || true)"
  [ -n "$http_port" ] || http_port=80
  [ -n "$https_port" ] || https_port=443

  if [ "$tls_mode" = "none" ] || [ -z "$tls_mode" ]; then
    probe="http://127.0.0.1:${http_port}/healthz"
  else
    probe="https://127.0.0.1:${https_port}/healthz"
  fi

  for _ in $(seq 1 45); do
    if curl -fsSk "$probe" >/dev/null 2>&1; then
      log "API healthy ($probe)"
      return 0
    fi
    sleep 2
  done
  err "API health probe failed ($probe)"
  return 1
}

rollback() {
  if [ "$ROLLBACK_DONE" -eq 1 ]; then
    return 0
  fi
  ROLLBACK_DONE=1

  if [ -z "$OLD_HEAD" ]; then
    warn "Rollback skipped: OLD_HEAD is empty"
    return 0
  fi

  warn "Rolling back to commit $OLD_HEAD"
  git reset --hard "$OLD_HEAD" >>"$LOG_FILE" 2>&1 || {
    err "Rollback git reset failed"
    return 1
  }

  warn "Rebuilding previous version containers"
  docker compose up -d --build --remove-orphans >>"$LOG_FILE" 2>&1 || {
    err "Rollback compose up failed"
    return 1
  }

  if probe_api_health; then
    warn "Rollback completed successfully"
  else
    err "Rollback completed but health check still failing"
  fi
}

on_error() {
  local rc=$?
  local line="$1"
  local cmd="$2"
  err "Update failed at line $line (exit $rc)"
  err "Last command: $cmd"
  rollback || true
  err "See log: $LOG_FILE"
  exit "$rc"
}
trap 'on_error $LINENO "$BASH_COMMAND"' ERR

main() {
  require_root
  require_env

  mkdir -p "$(dirname "$LOG_FILE")"
  touch "$LOG_FILE"

  log "Starting update in $INSTALL_DIR (branch=$REPO_BRANCH)"
  cd "$INSTALL_DIR"

  OLD_HEAD="$(git rev-parse HEAD)"
  log "Current commit: $OLD_HEAD"

  git fetch --prune --quiet origin "$REPO_BRANCH"
  NEW_HEAD="$(git rev-parse "origin/$REPO_BRANCH")"
  log "Target commit:  $NEW_HEAD"

  if [ "$OLD_HEAD" = "$NEW_HEAD" ]; then
    log "Repository already up-to-date"
  else
    git reset --hard "origin/$REPO_BRANCH" >>"$LOG_FILE" 2>&1
    log "Checked out latest origin/$REPO_BRANCH"

    log "Changes included in this update:"
    git --no-pager log --oneline --no-decorate "$OLD_HEAD..$NEW_HEAD" | sed 's/^/  - /' | tee -a "$LOG_FILE" || true
  fi

  # Keep user data untouched: .env, runtime/*, TLS certs are outside tracked changes.
  log "Pulling images"
  docker compose pull >>"$LOG_FILE" 2>&1 || warn "docker compose pull failed, continuing with local cache"

  log "Rebuilding and restarting stack"
  docker compose up -d --build --remove-orphans >>"$LOG_FILE" 2>&1

  probe_api_health

  log "Container status:"
  docker compose ps | tee -a "$LOG_FILE"

  log "Update completed successfully"
}

main "$@"

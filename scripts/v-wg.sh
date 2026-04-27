#!/usr/bin/env bash
set -Eeuo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/void-wg}"
COMPOSE="docker compose -f $INSTALL_DIR/docker-compose.yml"

usage() {
  cat <<'EOF'
Usage: v-wg <command>

Commands:
  status          Show stack status
  start           Start stack
  stop            Stop stack
  restart         Restart stack
  logs            Follow logs
  update          Update to latest version
  renew           Renew TLS certificate
  uninstall       Remove installation
  help            Show this help
EOF
}

require_root() {
  [ "$(id -u)" -eq 0 ] || { echo "run as root: sudo v-wg <command>" >&2; exit 1; }
}

require_install() {
  [ -f "$INSTALL_DIR/docker-compose.yml" ] || { echo "not installed at $INSTALL_DIR" >&2; exit 1; }
}

cmd="${1:-help}"

require_root

case "$cmd" in
  status)
    require_install
    $COMPOSE ps
    ;;
  start)
    require_install
    $COMPOSE up -d --remove-orphans
    ;;
  stop)
    require_install
    $COMPOSE stop
    ;;
  restart)
    require_install
    $COMPOSE restart
    ;;
  logs)
    require_install
    $COMPOSE logs -f --tail=100
    ;;
  update)
    require_install
    bash "$INSTALL_DIR/scripts/update.sh"
    ;;
  renew)
    require_install
    bash "$INSTALL_DIR/scripts/renew-cert.sh"
    ;;
  uninstall)
    require_install
    bash "$INSTALL_DIR/scripts/uninstall.sh"
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    echo "unknown command: $cmd" >&2
    usage
    exit 1
    ;;
esac


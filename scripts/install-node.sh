#!/usr/bin/env bash
# void-wg node installer.
#
# Запускается с --control-url=<url> --node-id=<uuid> --secret=<hex>.
# Этот скрипт ОТДАЁТСЯ control-plane'ом по адресу /install-node.sh
# и встроен в backend как const nodeInstallScript.
#
# Логика:
#   1) проверка root + установка docker / docker compose
#   2) скачивание исходников агента из <CONTROL_URL>/static/agent.tar.gz
#      (control-plane отдаёт публично; альтернатива — клонировать репозиторий)
#   3) локальный docker build → image void/node:latest
#   4) генерация /opt/void-node/docker-compose.yml
#   5) docker compose up -d
#
# Идемпотентен: повторный запуск пересобирает образ с актуальным кодом.
set -Eeuo pipefail

CONTROL_URL=""
NODE_ID=""
SECRET=""
NODE_VERSION="${NODE_VERSION:-latest}"
NODE_INSTALL_DIR="${NODE_INSTALL_DIR:-/opt/void-node}"
# REPO_URL — откуда тянуть исходники. По умолчанию резолвим из control-plane'а
# (репозиторий должен совпадать с тем, откуда установлена сама панель).
REPO_URL="${REPO_URL:-https://github.com/wester11/v-ui.git}"
REPO_BRANCH="${REPO_BRANCH:-main}"

for arg in "$@"; do
  case "$arg" in
    --control-url=*) CONTROL_URL="${arg#*=}" ;;
    --node-id=*)     NODE_ID="${arg#*=}" ;;
    --secret=*)      SECRET="${arg#*=}" ;;
    --repo=*)        REPO_URL="${arg#*=}" ;;
    --branch=*)      REPO_BRANCH="${arg#*=}" ;;
  esac
done

# Pretty-печать
G='\033[0;32m'; Y='\033[1;33m'; R='\033[0;31m'; N='\033[0m'
log()  { printf "${G}[node-install] %s${N}\n" "$*"; }
warn() { printf "${Y}[node-install] %s${N}\n" "$*"; }
err()  { printf "${R}[node-install] %s${N}\n" "$*" >&2; }
die()  { err "$*"; exit 1; }

[ "$(id -u)" -eq 0 ] || die "run as root"
[ -n "$CONTROL_URL" ] || die "--control-url is required"
[ -n "$NODE_ID" ]     || die "--node-id is required"
[ -n "$SECRET" ]      || die "--secret is required"

log "Installing prerequisites..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq ca-certificates curl gnupg lsb-release git

if ! command -v docker >/dev/null 2>&1; then
    log "Installing Docker Engine + Compose plugin..."
    install -m 0755 -d /etc/apt/keyrings
    OS_ID="$(. /etc/os-release; echo "$ID")"
    curl -fsSL "https://download.docker.com/linux/${OS_ID}/gpg" \
      | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${OS_ID} $(lsb_release -cs) stable" \
      > /etc/apt/sources.list.d/docker.list
    apt-get update -qq
    apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    systemctl enable --now docker
fi

mkdir -p "$NODE_INSTALL_DIR"

# Чистый shallow-clone (или pull, если уже есть).
SRC_DIR="$NODE_INSTALL_DIR/src"
if [ -d "$SRC_DIR/.git" ]; then
    log "Updating sources in $SRC_DIR..."
    git -C "$SRC_DIR" fetch --quiet origin "$REPO_BRANCH"
    git -C "$SRC_DIR" reset --hard "origin/$REPO_BRANCH"
else
    log "Cloning $REPO_URL ($REPO_BRANCH) -> $SRC_DIR..."
    rm -rf "$SRC_DIR"
    git clone --depth=1 --branch "$REPO_BRANCH" "$REPO_URL" "$SRC_DIR"
fi

log "Building void/node:${NODE_VERSION} from $SRC_DIR/agent..."
docker build -t "void/node:${NODE_VERSION}" "$SRC_DIR/agent" >/dev/null

cat >"$NODE_INSTALL_DIR/docker-compose.yml" <<EOF
services:
  void-node:
    image: void/node:${NODE_VERSION}
    container_name: void-node
    network_mode: host
    restart: always
    environment:
      - CONTROL_URL=${CONTROL_URL}
      - NODE_ID=${NODE_ID}
      - SECRET=${SECRET}
      - HTTP_LISTEN=:7443
      - AGENT_INSECURE_TLS=false
EOF

cd "$NODE_INSTALL_DIR"
docker compose up -d --remove-orphans

log "Node agent started. Verifying registration with control-plane..."
sleep 3
if curl -fsSk "${CONTROL_URL}/healthz" >/dev/null 2>&1; then
    log "Control-plane reachable at ${CONTROL_URL}"
else
    warn "Control-plane health-check failed — agent will retry in background"
fi

log "Node ${NODE_ID} is online. Tail agent logs:"
log "  docker logs -f void-node"

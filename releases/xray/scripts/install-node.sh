#!/usr/bin/env bash
# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║  void-wg  NODE INSTALLER                                                    ║
# ║  Hosted at: https://raw.githubusercontent.com/wester11/v-ui/main/           ║
# ║             scripts/install-node.sh                                         ║
# ║                                                                              ║
# ║  Usage (панель генерирует автоматически):                                   ║
# ║    bash <(curl -Ls https://raw.githubusercontent.com/wester11/v-ui/         ║
# ║           main/scripts/install-node.sh) BASE64_TOKEN                        ║
# ║                                                                              ║
# ║  TOKEN = base64("CONTROL_URL NODE_ID SECRET")                               ║
# ║                                                                              ║
# ║  Идемпотентен: повторный запуск обновляет и перезапускает агент.             ║
# ╚══════════════════════════════════════════════════════════════════════════════╝
set -Eeuo pipefail

# ── Цвета ────────────────────────────────────────────────────────────────────
B='\033[1m'; G='\033[0;32m'; Y='\033[1;33m'; R='\033[0;31m'
C='\033[0;36m'; M='\033[0;35m'; N='\033[0m'

log()     { printf "${G}${B}✓${N}${B} %s${N}\n" "$*"; }
info()    { printf "${C}  » %s${N}\n" "$*"; }
warn()    { printf "${Y}${B}⚠${N} %s${N}\n" "$*"; }
section() { printf "\n${M}${B}── %s ──${N}\n" "$*"; }
err()     { printf "${R}${B}✗ %s${N}\n" "$*" >&2; }
die()     { err "$*"; exit 1; }

# ── Баннер ───────────────────────────────────────────────────────────────────
printf "${M}${B}"
printf " __   __  ___  ___ ____     _  __ ____ \n"
printf " \ \ / / / _ \|_ _|  _ \   | | \ V  _  \ \n"
printf "  \ V / | | | || || | | |  | |  \ / | |\n"
printf "   \_/  |_| |_|___|_| |_|  |_|   \_/_| |\n"
printf "  Node Installer                         \n"
printf "${N}\n"

# ── Переменные окружения ──────────────────────────────────────────────────────
NODE_INSTALL_DIR="${NODE_INSTALL_DIR:-/opt/void-node}"
REPO_URL="${REPO_URL:-https://github.com/wester11/v-ui.git}"
REPO_BRANCH="${REPO_BRANCH:-main}"
NODE_IMAGE="${NODE_IMAGE:-void/node:latest}"

# ── Декодирование токена ──────────────────────────────────────────────────────
TOKEN="${1:-}"

CONTROL_URL=""; NODE_ID=""; SECRET=""

if [[ -n "$TOKEN" ]]; then
  # Новый формат: один base64-токен = "CONTROL_URL NODE_ID SECRET"
  decoded="$(echo "$TOKEN" | base64 -d 2>/dev/null)" \
    || die "Не удалось декодировать токен. Убедитесь что команда скопирована полностью."
  read -r CONTROL_URL NODE_ID SECRET <<< "$decoded"
else
  # Обратная совместимость: старые --флаги
  for arg in "$@"; do
    case "$arg" in
      --control-url=*) CONTROL_URL="${arg#*=}" ;;
      --node-id=*)     NODE_ID="${arg#*=}" ;;
      --secret=*)      SECRET="${arg#*=}" ;;
      --repo=*)        REPO_URL="${arg#*=}" ;;
      --branch=*)      REPO_BRANCH="${arg#*=}" ;;
    esac
  done
fi

# ── Валидация ─────────────────────────────────────────────────────────────────
[[ "$(id -u)" -eq 0 ]] || die "Запустите от root (sudo bash ...)"
[[ -n "$CONTROL_URL" ]] || die "CONTROL_URL не найден в токене"
[[ -n "$NODE_ID" ]]     || die "NODE_ID не найден в токене"
[[ -n "$SECRET" ]]      || die "SECRET не найден в токене"

section "Конфигурация"
info "Control URL : $CONTROL_URL"
info "Node ID     : $NODE_ID"
info "Install dir : $NODE_INSTALL_DIR"
info "Repo        : $REPO_URL ($REPO_BRANCH)"

# ── Зависимости ───────────────────────────────────────────────────────────────
section "Системные пакеты"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq 2>&1 | tail -1
apt-get install -y -qq ca-certificates curl gnupg lsb-release git 2>&1 | tail -1
log "Базовые пакеты установлены"

# ── Docker ───────────────────────────────────────────────────────────────────
section "Docker"
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  log "Docker уже установлен: $(docker --version)"
else
  info "Устанавливаем Docker Engine + Compose plugin..."
  install -m 0755 -d /etc/apt/keyrings
  OS_ID="$(. /etc/os-release; echo "$ID")"
  curl -fsSL "https://download.docker.com/linux/${OS_ID}/gpg" \
    | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/${OS_ID} $(lsb_release -cs) stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -qq
  apt-get install -y -qq \
    docker-ce docker-ce-cli containerd.io \
    docker-buildx-plugin docker-compose-plugin
  systemctl enable --now docker
  log "Docker $(docker --version) установлен"
fi

# ── Исходники агента ──────────────────────────────────────────────────────────
section "Исходники агента"
mkdir -p "$NODE_INSTALL_DIR"
SRC_DIR="$NODE_INSTALL_DIR/src"

if [[ -d "$SRC_DIR/.git" ]]; then
  info "Обновляем существующий репозиторий..."
  git -C "$SRC_DIR" remote set-url origin "$REPO_URL"
  git -C "$SRC_DIR" fetch --quiet origin "$REPO_BRANCH"
  git -C "$SRC_DIR" reset --hard "origin/$REPO_BRANCH"
else
  info "Клонируем $REPO_URL [$REPO_BRANCH]..."
  rm -rf "$SRC_DIR"
  git clone --depth=1 --branch "$REPO_BRANCH" "$REPO_URL" "$SRC_DIR" \
    || die "Не удалось клонировать репозиторий. Проверьте REPO_URL."
fi
log "Исходники получены: $(git -C "$SRC_DIR" log -1 --format='%h %s')"

# ── Сборка образа ─────────────────────────────────────────────────────────────
section "Docker image"
AGENT_DIR="$SRC_DIR/agent"
[[ -d "$AGENT_DIR" ]] || die "Директория $AGENT_DIR не найдена. Убедитесь, что репозиторий содержит папку agent/."

info "Собираем $NODE_IMAGE..."
docker build -t "$NODE_IMAGE" "$AGENT_DIR"
log "Образ $NODE_IMAGE собран"

# ── docker-compose.yml ────────────────────────────────────────────────────────
section "Конфигурация агента"
cat > "$NODE_INSTALL_DIR/docker-compose.yml" << EOF
services:
  void-node:
    image: ${NODE_IMAGE}
    container_name: void-node
    network_mode: host
    restart: always
    environment:
      CONTROL_URL:        ${CONTROL_URL}
      NODE_ID:            ${NODE_ID}
      SECRET:             ${SECRET}
      HTTP_LISTEN:        :7443
      AGENT_INSECURE_TLS: "true"
EOF
log "docker-compose.yml создан в $NODE_INSTALL_DIR"

# ── Запуск ────────────────────────────────────────────────────────────────────
section "Запуск агента"
cd "$NODE_INSTALL_DIR"
docker compose down --remove-orphans 2>/dev/null || true
docker compose up -d
log "Агент запущен"

# ── Проверка связи с панелью ──────────────────────────────────────────────────
section "Проверка связи"
info "Ожидаем регистрацию в панели (до 15 секунд)..."
for i in $(seq 1 5); do
  sleep 3
  # -k = игнорировать самоподписанный сертификат панели (нормально для IP-адреса)
  if curl -fsk "${CONTROL_URL}/healthz" >/dev/null 2>&1; then
    log "Панель доступна: ${CONTROL_URL}"
    break
  fi
  info "Попытка $i/5..."
done

# ── Готово ────────────────────────────────────────────────────────────────────
printf "\n${G}${B}══════════════════════════════════════════════${N}\n"
printf "${G}${B}  Нода успешно установлена и запущена! 🎉${N}\n"
printf "${G}${B}══════════════════════════════════════════════${N}\n\n"
info "Node ID    : $NODE_ID"
info "Логи агента: docker logs -f void-node"
info "Перезапуск : cd $NODE_INSTALL_DIR && docker compose restart"
info "Удалить    : cd $NODE_INSTALL_DIR && docker compose down"
printf "\n"

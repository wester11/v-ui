#!/usr/bin/env bash
# void-wg one-click installer.
#
#   bash <(curl -Ls https://raw.githubusercontent.com/wester11/v-ui/main/scripts/install.sh)
#
# Идемпотентен: повторный запуск не ломает уже установленную копию.
# Логи: /var/log/void-wg-install.log

set -Eeuo pipefail

# DEBUG=1 — включает trace + verbose-логирование команд.
DEBUG="${DEBUG:-0}"
if [ "$DEBUG" = "1" ]; then
    export PS4='+ [${BASH_SOURCE##*/}:${LINENO}] '
    set -x
fi

# ----- defaults / overrides via env -----
REPO_URL="${REPO_URL:-https://github.com/wester11/v-ui.git}"
REPO_BRANCH="${REPO_BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/void-wg}"
PANEL_HTTP_PORT="${PANEL_HTTP_PORT:-80}"
PANEL_HTTPS_PORT_INPUT="${PANEL_HTTPS_PORT:-}"
PANEL_HTTPS_PORT="${PANEL_HTTPS_PORT:-443}"
PANEL_RANDOM_HTTPS_PORT="${PANEL_RANDOM_HTTPS_PORT:-0}"
WG_PORT="${WG_PORT:-51820}"
OBFS_PORT="${OBFS_PORT:-51821}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@local}"
LOG_FILE="${LOG_FILE:-/var/log/void-wg-install.log}"

# TLS — опциональные env'ы для не-интерактивного режима:
#   TLS_MODE=selfsigned                              — IP, self-signed
#   TLS_MODE=letsencrypt PANEL_DOMAIN=vpn.example.com — Let's Encrypt
TLS_MODE="${TLS_MODE:-}"
PANEL_DOMAIN="${PANEL_DOMAIN:-}"
PANEL_ENTRY_TOKEN="${PANEL_ENTRY_TOKEN:-}"
PANEL_HTTPS_PORT_RANDOMIZED="${PANEL_HTTPS_PORT_RANDOMIZED:-}"

# ----- pretty -----
# Enable ANSI colors only when writing to a TTY.
if [ -t 1 ]; then
    GREEN=$'\033[0;32m'
    YELLOW=$'\033[1;33m'
    RED=$'\033[0;31m'
    CYAN=$'\033[0;36m'
    BOLD=$'\033[1m'
    NC=$'\033[0m'
else
    GREEN=''
    YELLOW=''
    RED=''
    CYAN=''
    BOLD=''
    NC=''
fi

ts()    { date '+%Y-%m-%d %H:%M:%S'; }
log()   { printf "${GREEN}[%s] [INFO] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
ok()    { printf "${GREEN}[%s] [ OK ] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
warn()  { printf "${YELLOW}[%s] [WARN] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
err()   { printf "${RED}[%s] [ERROR] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE" >&2; }
die()   { err "$*"; exit 1; }

TOTAL_STEPS=8
step()  { printf "\n${CYAN}[STEP %d/%d] %s${NC}\n" "$1" "$TOTAL_STEPS" "$2" | tee -a "$LOG_FILE"; }
dbg()   { [ "$DEBUG" = "1" ] && printf "[DEBUG] %s\n" "$*" | tee -a "$LOG_FILE" >&2 || true; }
hint()  { printf "${YELLOW}  • %s${NC}\n" "$*" | tee -a "$LOG_FILE" >&2; }

# run — выполняет команду с логированием; stdout/stderr → лог-файл.
# При ошибке возвращает RC, чтобы trap мог сработать выше по стеку.
run() {
    dbg "Running: $*"
    if [ "$DEBUG" = "1" ]; then
        # В debug-режиме видим вывод и в консоли, и в логе.
        "$@" 2>&1 | tee -a "$LOG_FILE"
        local rc=${PIPESTATUS[0]}
    else
        "$@" >>"$LOG_FILE" 2>&1
        local rc=$?
    fi
    dbg "Exit code: $rc"
    return $rc
}

# runq — то же, что run, но без падения на non-zero (для опциональных команд).
runq() {
    dbg "Running (best-effort): $*"
    "$@" >>"$LOG_FILE" 2>&1 || true
}

apt_run() {
    run env DEBIAN_FRONTEND=noninteractive apt-get \
        -o DPkg::Lock::Timeout=120 \
        -o Acquire::Retries=3 \
        "$@"
}

# dump_recent_log — последние 50 строк install-лога (для on_error / диагностики).
dump_recent_log() {
    local n="${1:-50}"
    if [ -f "$LOG_FILE" ]; then
        printf "${YELLOW}--- last %d lines of %s ---${NC}\n" "$n" "$LOG_FILE" >&2
        tail -n "$n" "$LOG_FILE" >&2 || true
        printf "${YELLOW}--- end of log tail ---${NC}\n" >&2
    fi
}

# dump_compose_logs — выводит ps + последние строки логов всех сервисов.
# Безопасно вызывать даже если стек ещё не поднимался.
dump_compose_logs() {
    local n="${1:-50}"
    [ -f "$INSTALL_DIR/docker-compose.yml" ] || return 0
    command -v docker >/dev/null 2>&1 || return 0
    printf "${YELLOW}--- docker compose ps ---${NC}\n" >&2
    ( cd "$INSTALL_DIR" && docker compose ps 2>&1 ) >&2 || true
    printf "${YELLOW}--- docker compose logs (last %d lines) ---${NC}\n" "$n" >&2
    ( cd "$INSTALL_DIR" && docker compose logs --tail="$n" 2>&1 ) >&2 || true
}

# port_in_use — печатает кто занимает порт (lsof / ss fallback).
port_in_use() {
    local port="$1"
    if command -v lsof >/dev/null 2>&1; then
        lsof -nP -i ":${port}" 2>/dev/null || true
    elif command -v ss >/dev/null 2>&1; then
        ss -tulnp "sport = :${port}" 2>/dev/null || true
    fi
}

on_error() {
    local rc=$?
    local line="$1"
    local cmd="$2"
    err "Installation failed at line $line (exit $rc)"
    err "Last command: $cmd"
    dump_recent_log 50
    # Если на момент падения стек уже поднимался — добавляем compose-диагностику.
    case "$cmd" in
        *docker*compose*|*"compose up"*|*"compose pull"*|*"compose build"*)
            dump_compose_logs 50
            ;;
    esac
    err "Full log: $LOG_FILE"
    exit "$rc"
}
trap 'on_error $LINENO "$BASH_COMMAND"' ERR

# read из терминала, даже если stdin = process substitution (bash <(curl ...))
ask() {
    local prompt="$1" varname="$2" default="${3:-}" reply
    if [ -n "$default" ]; then prompt="$prompt [$default]"; fi
    if [ -t 0 ] || [ -e /dev/tty ]; then
        read -r -p "$prompt: " reply < /dev/tty || reply=""
    else
        reply=""
    fi
    [ -z "$reply" ] && reply="$default"
    printf -v "$varname" '%s' "$reply"
}

confirm_yes_default() {
    local prompt="$1" reply
    if [ -t 0 ] || [ -e /dev/tty ]; then
        read -r -p "$prompt [Y/n]: " reply < /dev/tty || reply=""
    else
        reply=""
    fi
    case "${reply,,}" in
        ""|y|yes) return 0 ;;
        *) return 1 ;;
    esac
}

# ----- steps -----
require_root() {
    if [ "$(id -u)" -ne 0 ]; then
        die "This script must be run as root.  Try:  sudo bash $0"
    fi
}

detect_os() {
    [ -f /etc/os-release ] || die "/etc/os-release missing — unsupported OS"
    # shellcheck disable=SC1091
    . /etc/os-release
    OS_ID="${ID:-unknown}"
    OS_MAJOR="${VERSION_ID%%.*}"
    case "$OS_ID" in
        ubuntu)
            [ "${OS_MAJOR:-0}" -ge 20 ] || die "Ubuntu 20.04+ required (you have ${VERSION_ID:-?})"
            ;;
        debian)
            [ "${OS_MAJOR:-0}" -ge 11 ] || die "Debian 11+ required (you have ${VERSION_ID:-?})"
            ;;
        *)
            die "Unsupported OS: $OS_ID. Supported: Ubuntu 20.04+, Debian 11+"
            ;;
    esac
    log "OS detected: ${PRETTY_NAME:-$OS_ID $VERSION_ID}"
}

install_apt_packages() {
    log "Refreshing apt cache..."
    apt_run update -qq \
        || die "apt-get update failed (см. $LOG_FILE). Возможные причины: нет интернета, сломан /etc/apt/sources.list, заблокирован lock /var/lib/dpkg/lock-frontend другим apt-процессом."
    log "Installing prerequisites: git, curl, openssl, ufw, wireguard-tools, iptables, certbot, dnsutils, lsof..."
    if ! apt_run install -y -qq \
        ca-certificates curl gnupg lsb-release git openssl ufw \
        wireguard-tools iptables jq certbot dnsutils lsof; then
        warn "apt-get install failed; attempting automatic recovery (dpkg/apt fix) and retry..."
        runq dpkg --configure -a
        runq env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=120 -f install -y -qq
        runq env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=120 update -qq
        apt_run install -y -qq \
            ca-certificates curl gnupg lsb-release git openssl ufw \
            wireguard-tools iptables jq certbot dnsutils lsof \
            || { dump_recent_log 120; die "apt-get install failed after recovery attempt (см. $LOG_FILE). Попробуйте вручную: dpkg --configure -a && apt-get -f install -y && apt-get install -y certbot dnsutils"; }
    fi
}

install_docker() {
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        log "Docker already installed: $(docker --version | head -n1)"
        return
    fi
    log "Installing Docker Engine + Compose plugin..."
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL "https://download.docker.com/linux/${OS_ID}/gpg" \
        | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
        || die "Failed to download Docker GPG key (no internet?)"
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${OS_ID} $(lsb_release -cs) stable" \
        > /etc/apt/sources.list.d/docker.list
    apt_run update -qq
    apt_run install -y -qq \
        docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin \
        || die "Docker install failed (см. $LOG_FILE)."
    runq systemctl enable --now docker
    log "Docker installed: $(docker --version | head -n1)"
}

clone_repo() {
    if [ -d "$INSTALL_DIR/.git" ]; then
        log "Repo already present at $INSTALL_DIR — pulling latest"
        local current_origin
        current_origin="$(git -C "$INSTALL_DIR" remote get-url origin 2>/dev/null || true)"
        if [ "$current_origin" != "$REPO_URL" ]; then
            log "Updating git origin: ${current_origin:-<none>} -> $REPO_URL"
            run git -C "$INSTALL_DIR" remote set-url origin "$REPO_URL" \
                || die "git remote set-url failed: $REPO_URL"
        fi
        run git -C "$INSTALL_DIR" fetch --quiet origin "$REPO_BRANCH" \
            || die "git fetch failed: проверьте сетевой доступ к $REPO_URL"
        run git -C "$INSTALL_DIR" reset --hard "origin/$REPO_BRANCH" \
            || die "git reset failed (локальные изменения в $INSTALL_DIR?)"
    else
        log "Cloning $REPO_URL ($REPO_BRANCH) -> $INSTALL_DIR"
        rm -rf "$INSTALL_DIR"
        run git clone --depth=1 --branch "$REPO_BRANCH" "$REPO_URL" "$INSTALL_DIR" \
            || die "git clone failed: $REPO_URL ($REPO_BRANCH) — проверьте URL и доступ."
    fi
    mkdir -p "$INSTALL_DIR/runtime/tls" "$INSTALL_DIR/runtime/acme-www" "$INSTALL_DIR/runtime/agent-ca"
    # API runs as non-root user (uid/gid 65532) in container; ensure mTLS dir is writable.
    if ! chown -R 65532:65532 "$INSTALL_DIR/runtime/agent-ca" 2>/dev/null; then
        warn "Could not chown runtime/agent-ca to 65532:65532; falling back to chmod 0777"
        chmod -R 0777 "$INSTALL_DIR/runtime/agent-ca" || true
    fi
}

random_pass() {
    # Avoid SIGPIPE under `set -o pipefail` (no pipeline here).
    openssl rand -hex 8
}

ensure_env_file() {
    local env_file="$INSTALL_DIR/.env"
    if [ -f "$env_file" ] && grep -q '^BOOTSTRAP_ADMIN_PASSWORD=' "$env_file"; then
        log ".env exists — keeping current credentials"
        # shellcheck disable=SC1090
        set -a; . "$env_file"; set +a
        return
    fi
    log "Generating .env with fresh secrets..."
    JWT_SECRET="$(openssl rand -hex 32)"
    BOOTSTRAP_ADMIN_PASSWORD="$(random_pass)"
    cat > "$env_file" <<EOF
JWT_SECRET=$JWT_SECRET
BOOTSTRAP_ADMIN_EMAIL=$ADMIN_EMAIL
BOOTSTRAP_ADMIN_PASSWORD=$BOOTSTRAP_ADMIN_PASSWORD
PANEL_HTTP_PORT=$PANEL_HTTP_PORT
PANEL_HTTPS_PORT=$PANEL_HTTPS_PORT
WG_PORT=$WG_PORT
OBFS_PORT=$OBFS_PORT
LOG_LEVEL=info
AGENT_INSECURE_TLS=false
MTLS_DIR=/opt/void-wg/runtime/agent-ca
PUBLIC_BASE_URL=
EOF
    chmod 600 "$env_file"
    export BOOTSTRAP_ADMIN_PASSWORD ADMIN_EMAIL
}

# Записать/обновить ключ=значение в .env (idempotent)
env_set() {
    local key="$1" value="$2" file="$INSTALL_DIR/.env"
    if grep -q "^${key}=" "$file" 2>/dev/null; then
        sed -i "s|^${key}=.*|${key}=${value}|" "$file"
    else
        printf '%s=%s\n' "$key" "$value" >> "$file"
    fi
}

public_ip() {
    local ip
    ip="$(curl -fsS -4 https://ifconfig.me 2>/dev/null || true)"
    [ -n "$ip" ] || ip="$(curl -fsS -4 https://ifconfig.io 2>/dev/null || true)"
    [ -n "$ip" ] || ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
    [ -n "$ip" ] || ip="SERVER_IP"
    echo "$ip"
}

resolve_domain() {
    local domain="$1"
    if command -v dig >/dev/null 2>&1; then
        dig +short "$domain" A 2>/dev/null | grep -E '^[0-9.]+$' | tail -n1
    else
        getent ahostsv4 "$domain" 2>/dev/null | awk 'NR==1{print $1}'
    fi
}

# ===== TLS configuration =====

choose_tls_mode_interactive() {
    if [ -n "$TLS_MODE" ]; then
        log "TLS_MODE=$TLS_MODE (from env)"
        return
    fi
    printf "\n${BOLD}SSL setup${NC}\n"
    printf "  ${BOLD}1)${NC} ${GREEN}Domain (Let's Encrypt)${NC}  ${BOLD}[recommended]${NC}\n"
    printf "     - trusted cert, no browser warnings\n"
    printf "     - automatic renewal\n"
    printf "  ${BOLD}2)${NC} ${GREEN}IP address (self-signed)${NC}\n"
    printf "     - works immediately\n"
    printf "     - browser warning expected\n\n"

    local choice
    while true; do
        ask "Select [1-2]" choice "2"
        case "$choice" in
            1) TLS_MODE=letsencrypt; return ;;
            2|"") TLS_MODE=selfsigned; return ;;
            *) warn "Please enter 1 or 2." ;;
        esac
    done
}

pick_random_https_port() {
    local p tries=0
    while [ "$tries" -lt 64 ]; do
        p=$(( (RANDOM % 45536) + 20000 ))
        if ! port_in_use "$p" | grep -q .; then
            echo "$p"
            return 0
        fi
        tries=$((tries + 1))
    done
    # Fallback to a known default if no free random port found.
    echo "443"
}

ensure_panel_access_endpoint() {
    if [ -z "$PANEL_ENTRY_TOKEN" ]; then
        PANEL_ENTRY_TOKEN="$(openssl rand -hex 16)"
    fi

    # Randomize HTTPS host port only when explicitly requested.
    # Default behavior is stable production port 443.
    if [ -z "$PANEL_HTTPS_PORT_RANDOMIZED" ]; then
        case "${PANEL_RANDOM_HTTPS_PORT,,}" in
            1|true|yes)
                if [ -z "$PANEL_HTTPS_PORT_INPUT" ] && [ "${PANEL_HTTPS_PORT:-443}" = "443" ]; then
                    PANEL_HTTPS_PORT="$(pick_random_https_port)"
                    log "Using randomized HTTPS port: $PANEL_HTTPS_PORT"
                fi
                PANEL_HTTPS_PORT_RANDOMIZED="true"
                ;;
            *)
                PANEL_HTTPS_PORT_RANDOMIZED="false"
                PANEL_HTTPS_PORT="${PANEL_HTTPS_PORT:-443}"
                ;;
        esac
    fi

    # Backward-compatibility: old installs may have randomized port persisted in .env.
    # Unless random mode is explicitly requested again, normalize back to 443.
    if [ "$PANEL_HTTPS_PORT_RANDOMIZED" = "true" ]; then
        case "${PANEL_RANDOM_HTTPS_PORT,,}" in
            1|true|yes) ;;
            *)
                if [ -z "$PANEL_HTTPS_PORT_INPUT" ]; then
                    PANEL_HTTPS_PORT="443"
                    PANEL_HTTPS_PORT_RANDOMIZED="false"
                    log "Random HTTPS port disabled; switching panel back to 443"
                fi
                ;;
        esac
    fi

    if [ -n "$PANEL_HTTPS_PORT_INPUT" ]; then
        PANEL_HTTPS_PORT="$PANEL_HTTPS_PORT_INPUT"
        PANEL_HTTPS_PORT_RANDOMIZED="false"
        log "Using explicit HTTPS port: $PANEL_HTTPS_PORT"
    fi

    if [ "$PANEL_HTTPS_PORT_RANDOMIZED" = "true" ] && [ "$PANEL_HTTPS_PORT" != "443" ]; then
        warn "Random HTTPS port is enabled. Ensure provider/cloud firewall allows TCP/$PANEL_HTTPS_PORT."
    fi

    env_set "PANEL_HTTPS_PORT" "$PANEL_HTTPS_PORT"
    env_set "PANEL_ENTRY_TOKEN" "$PANEL_ENTRY_TOKEN"
    env_set "PANEL_HTTPS_PORT_RANDOMIZED" "$PANEL_HTTPS_PORT_RANDOMIZED"
    env_set "PANEL_RANDOM_HTTPS_PORT" "$PANEL_RANDOM_HTTPS_PORT"
}

ask_domain_and_validate() {
    [ "$TLS_MODE" = "letsencrypt" ] || return 0

    if [ -z "$PANEL_DOMAIN" ]; then
        echo
        ask "Enter your domain (e.g. vpn.example.com)" PANEL_DOMAIN
        [ -n "$PANEL_DOMAIN" ] || die "Domain is required for Let's Encrypt"
    fi

    # Подтверждение выбора
    echo
    printf "  ${BOLD}You selected:${NC} Domain\n"
    printf "  ${BOLD}Domain:${NC}       %s\n\n" "$PANEL_DOMAIN"
    if ! confirm_yes_default "Continue?"; then
        die "Aborted by user"
    fi
}

confirm_ip_choice() {
    [ "$TLS_MODE" = "selfsigned" ] || return 0
    local ip; ip="$(public_ip)"
    echo
    printf "  ${BOLD}You selected:${NC} IP Address\n"
    printf "  ${BOLD}IP:${NC}           %s (self-signed, 10 years)\n\n" "$ip"
    if ! confirm_yes_default "Continue?"; then
        die "Aborted by user"
    fi
}

# Проверка DNS — обязательная для letsencrypt: если не сходится → выходим.
check_dns_strict() {
    [ "$TLS_MODE" = "letsencrypt" ] || return 0
    log "Checking domain DNS..."
    local server_ip resolved
    server_ip="$(public_ip)"
    resolved="$(resolve_domain "$PANEL_DOMAIN")"

    if [ -z "$resolved" ]; then
        err "Domain $PANEL_DOMAIN does not resolve to any IP."
        err "Please add A-record pointing to $server_ip and try again."
        exit 1
    fi
    if [ "$resolved" != "$server_ip" ]; then
        err "Domain does not point to this server IP"
        err "Expected: $server_ip"
        err "Got:      $resolved"
        err "Please fix DNS A record and try again."
        exit 1
    fi
    ok "Domain resolves correctly ($PANEL_DOMAIN -> $resolved)"
}

generate_selfsigned() {
    local ip; ip="$(public_ip)"
    local tls_dir="$INSTALL_DIR/runtime/tls"
    mkdir -p "$tls_dir"
    log "Generating self-signed certificate for IP $ip (10 years)..."
    openssl req -x509 -nodes -newkey rsa:2048 -days 3650 \
        -keyout "$tls_dir/privkey.pem" \
        -out    "$tls_dir/fullchain.pem" \
        -subj   "/CN=$ip" \
        -addext "subjectAltName=IP:$ip,DNS:localhost" \
        >>"$LOG_FILE" 2>&1
    chmod 600 "$tls_dir/privkey.pem"
    PANEL_DOMAIN="$ip"
    ok "Self-signed certificate created"
}

issue_letsencrypt() {
    local domain="$PANEL_DOMAIN"
    local tls_dir="$INSTALL_DIR/runtime/tls"
    local le_dir="/etc/letsencrypt/live/$domain"
    mkdir -p "$tls_dir"

    if [ -f "$le_dir/fullchain.pem" ] && [ -f "$le_dir/privkey.pem" ]; then
        ok "Let's Encrypt cert for $domain already present — skipping issuance"
    else
        log "Stopping anything bound to :80 (for ACME http-01)..."
        ( cd "$INSTALL_DIR" && runq docker compose stop frontend )
        runq systemctl stop nginx
        runq systemctl stop apache2

        # Pre-flight: убедиться, что :80 действительно свободен
        local p80_users
        p80_users="$(port_in_use 80)"
        if [ -n "$p80_users" ]; then
            err "Port 80 is still busy after stop attempt:"
            printf '%s\n' "$p80_users" | tee -a "$LOG_FILE" >&2
            hint "Manually stop whatever uses port 80 and re-run installer."
            hint "Example: sudo fuser -k 80/tcp"
            exit 1
        fi

        log "Requesting Let's Encrypt certificate..."
        if ! run certbot certonly \
                --standalone \
                --non-interactive \
                --agree-tos \
                --register-unsafely-without-email \
                --preferred-challenges http \
                -d "$domain"; then
            err "certbot failed"
            err "Possible reasons:"
            hint "Port 80 закрыт фаерволом провайдера (нужно открыть TCP/80 inbound)"
            hint "DNS A-record указывает не на этот сервер"
            hint "Rate-limit Let's Encrypt (5 неудач/час): подождите час или добавьте --staging"
            hint "Smth bound to :80: $(port_in_use 80 | head -n3 | tr '\n' '; ')"
            exit 1
        fi

        ok "Certificate issued successfully"
    fi

    log "Copying certificate into runtime/tls..."
    cp -L "$le_dir/fullchain.pem" "$tls_dir/fullchain.pem"
    cp -L "$le_dir/privkey.pem"   "$tls_dir/privkey.pem"
    chmod 600 "$tls_dir/privkey.pem"
}

write_runtime_nginx_conf() {
    local out="$INSTALL_DIR/runtime/nginx.conf"
    log "Writing HTTPS nginx config (server_name=$PANEL_DOMAIN)"
    sed -e "s|__SERVER_NAME__|${PANEL_DOMAIN}|g" \
        -e "s|__HTTPS_PORT__|${PANEL_HTTPS_PORT}|g" \
        -e "s|__PANEL_ENTRY_TOKEN__|${PANEL_ENTRY_TOKEN}|g" \
        "$INSTALL_DIR/frontend/nginx.https.conf.tpl" > "$out"
}

configure_tls() {
    choose_tls_mode_interactive
    case "$TLS_MODE" in
        letsencrypt)
            ask_domain_and_validate
            check_dns_strict
            issue_letsencrypt
            ;;
        selfsigned)
            confirm_ip_choice
            generate_selfsigned
            ;;
        *) die "Unknown TLS_MODE: $TLS_MODE" ;;
    esac

    ensure_panel_access_endpoint
    write_runtime_nginx_conf

    env_set "TLS_MODE"     "$TLS_MODE"
    env_set "PANEL_DOMAIN" "$PANEL_DOMAIN"

    local base_url="https://${PANEL_DOMAIN}"
    [ "${PANEL_HTTPS_PORT}" = "443" ] || base_url="${base_url}:${PANEL_HTTPS_PORT}"
    env_set "PUBLIC_BASE_URL" "$base_url"
}

configure_firewall() {
    if ! command -v ufw >/dev/null 2>&1; then
        warn "ufw not installed — skipping firewall step"
        return
    fi
    log "Configuring ufw (allow 22, $PANEL_HTTP_PORT/tcp, $PANEL_HTTPS_PORT/tcp, $WG_PORT/udp, $OBFS_PORT/udp)"
    ufw --force default deny incoming >/dev/null
    ufw --force default allow outgoing >/dev/null
    ufw allow 22/tcp >/dev/null
    ufw allow "${PANEL_HTTP_PORT}/tcp"  >/dev/null
    ufw allow "${PANEL_HTTPS_PORT}/tcp" >/dev/null
    ufw allow "${WG_PORT}/udp"   >/dev/null
    ufw allow "${OBFS_PORT}/udp" >/dev/null
    ufw --force enable >/dev/null
}

install_systemd_unit() {
    local unit_src="$INSTALL_DIR/deploy/void-wg.service"
    local unit_dst=/etc/systemd/system/void-wg.service
    if [ ! -f "$unit_src" ]; then
        warn "$unit_src missing — skipping systemd setup"
        return
    fi
    log "Installing systemd unit: $unit_dst"
    sed "s|/opt/void-wg|$INSTALL_DIR|g" "$unit_src" > "$unit_dst"
    systemctl daemon-reload
    runq systemctl enable void-wg.service
}

install_cli_wrapper() {
    local src="$INSTALL_DIR/scripts/v-wg.sh"
    local dst="/usr/local/bin/v-wg"
    if [ ! -f "$src" ]; then
        warn "$src missing — skipping v-wg CLI install"
        return
    fi
    log "Installing v-wg CLI wrapper -> $dst"
    chmod +x "$src"
    ln -sfn "$src" "$dst"
}

install_renew_timer() {
    [ "$TLS_MODE" = "letsencrypt" ] || { log "TLS renewal timer not needed for mode=$TLS_MODE"; return; }
    local svc_src="$INSTALL_DIR/deploy/void-wg-renew.service"
    local tmr_src="$INSTALL_DIR/deploy/void-wg-renew.timer"
    [ -f "$svc_src" ] || { warn "$svc_src missing — skipping renew timer"; return; }
    [ -f "$tmr_src" ] || { warn "$tmr_src missing — skipping renew timer"; return; }
    log "Installing systemd renewal timer (twice daily)"
    sed "s|/opt/void-wg|$INSTALL_DIR|g" "$svc_src" > /etc/systemd/system/void-wg-renew.service
    cp "$tmr_src" /etc/systemd/system/void-wg-renew.timer
    systemctl daemon-reload
    run systemctl enable --now void-wg-renew.timer
    ok "Auto-renewal enabled"
}

install_update_timer() {
    local svc_src="$INSTALL_DIR/deploy/void-wg-update.service"
    local tmr_src="$INSTALL_DIR/deploy/void-wg-update.timer"
    [ -f "$svc_src" ] || { warn "$svc_src missing — skipping update timer"; return; }
    [ -f "$tmr_src" ] || { warn "$tmr_src missing — skipping update timer"; return; }
    log "Installing systemd update timer (daily)"
    sed "s|/opt/void-wg|$INSTALL_DIR|g" "$svc_src" > /etc/systemd/system/void-wg-update.service
    cp "$tmr_src" /etc/systemd/system/void-wg-update.timer
    systemctl daemon-reload
    run systemctl enable --now void-wg-update.timer
    ok "Auto-update enabled"
}

start_stack() {
    log "Building and starting containers (first run can take a few minutes)..."
    cd "$INSTALL_DIR"
    runq docker compose pull --quiet
    if ! run docker compose up -d --build --remove-orphans; then
        err "docker compose up failed"
        dump_recent_log 120
        dump_compose_logs 100
        hint "Run manually for live output: cd $INSTALL_DIR && docker compose up --build"
        hint "Если падает на build — проверьте, есть ли свободное место: df -h"
        exit 1
    fi

    log "Waiting for API to be ready..."
    local probe_url="https://127.0.0.1:${PANEL_HTTPS_PORT}/healthz"
    local ready=0
    for _ in $(seq 1 60); do
        if curl -fsSk "$probe_url" >/dev/null 2>&1; then
            ready=1; break
        fi
        sleep 2
    done
    if [ "$ready" -ne 1 ]; then
        warn "API did not become healthy within 120s — собираю диагностику..."
        dump_compose_logs 100
        local svc state bad=0
        for svc in postgres api frontend; do
            state="$(docker compose ps --format '{{.State}}' "$svc" 2>/dev/null | head -n1 || true)"
            if [ "$state" != "running" ]; then
                err "service $svc is in state: ${state:-unknown}"
                bad=1
            fi
        done
        hint "Tail logs in foreground: docker compose -f $INSTALL_DIR/docker-compose.yml logs -f"
        hint "Restart only api: docker compose -f $INSTALL_DIR/docker-compose.yml restart api"
        hint "TLS-cert problem? Check: ls -la $INSTALL_DIR/runtime/tls/"
        if [ "$bad" -ne 0 ]; then
            die "Stack is not healthy after startup; see logs above."
        fi
    else
        ok "API is healthy ($probe_url)"
    fi
}

print_summary() {
    local url
    case "$TLS_MODE" in
        letsencrypt) url="https://${PANEL_DOMAIN}" ;;
        selfsigned)  url="https://$(public_ip)" ;;
    esac
    [ "$PANEL_HTTPS_PORT" = "443" ] || url="$url:$PANEL_HTTPS_PORT"
    url="$url/$PANEL_ENTRY_TOKEN"

    cat <<SUMEOF

  Installation complete!

  Panel:    ${url}
  TLS mode: ${TLS_MODE}
  Login:    ${ADMIN_EMAIL}
  Password: ${BOOTSTRAP_ADMIN_PASSWORD}

  Files:    ${INSTALL_DIR}
  .env:     ${INSTALL_DIR}/.env  (mode 600)
  TLS dir:  ${INSTALL_DIR}/runtime/tls
  Logs:     docker compose -f ${INSTALL_DIR}/docker-compose.yml logs -f
  Update:   sudo bash ${INSTALL_DIR}/scripts/update.sh
  Auto-upd: systemctl status void-wg-update.timer
  Renew:    sudo bash ${INSTALL_DIR}/scripts/renew-cert.sh
  Remove:   sudo bash ${INSTALL_DIR}/scripts/uninstall.sh

  Manage:   sudo v-wg

  NOTE: Open panel only via the secret URL above (token path required).
  Save these credentials — they are also stored in ${INSTALL_DIR}/.env
SUMEOF
}

main() {
    require_root
    mkdir -p "$(dirname "$LOG_FILE")"
    : > "$LOG_FILE"
    log "void-wg installer starting..."
    if [ "$DEBUG" = "1" ]; then
        warn "DEBUG=1 — verbose output enabled (set -x + command tracing)"
    else
        log "tip: re-run with DEBUG=1 for verbose output"
    fi

    step 1 "Detecting OS and installing prerequisites"
    detect_os
    install_apt_packages

    step 2 "Installing Docker"
    install_docker

    step 3 "Cloning repository"
    clone_repo
    ensure_env_file

    step 4 "Setting up SSL"
    configure_tls

    step 5 "Configuring firewall"
    configure_firewall

    step 6 "Installing systemd units and CLI"
    install_systemd_unit
    install_renew_timer
    install_update_timer
    install_cli_wrapper

    step 7 "Starting containers"
    start_stack

    step 8 "Done"
    print_summary
}

main "$@"

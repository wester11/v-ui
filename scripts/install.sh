#!/usr/bin/env bash
# void-wg one-click installer.
#
#   bash <(curl -Ls https://raw.githubusercontent.com/wester11/void_wg/main/scripts/install.sh)
#
# Идемпотентен: повторный запуск не ломает уже установленную копию.
# Логи: /var/log/void-wg-install.log

set -Eeuo pipefail

# ----- defaults / overrides via env -----
REPO_URL="${REPO_URL:-https://github.com/wester11/void_wg.git}"
REPO_BRANCH="${REPO_BRANCH:-main}"
INSTALL_DIR="${INSTALL_DIR:-/opt/void-wg}"
PANEL_HTTP_PORT="${PANEL_HTTP_PORT:-80}"
PANEL_HTTPS_PORT="${PANEL_HTTPS_PORT:-443}"
WG_PORT="${WG_PORT:-51820}"
OBFS_PORT="${OBFS_PORT:-51821}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@local}"
LOG_FILE="${LOG_FILE:-/var/log/void-wg-install.log}"

# TLS — может задаваться через env (для не-интерактивного режима):
#   TLS_MODE=selfsigned                                — самоподписанный по IP
#   TLS_MODE=letsencrypt PANEL_DOMAIN=vpn.example.com LE_EMAIL=ops@example.com
#   TLS_MODE=none                                       — HTTP-only
TLS_MODE="${TLS_MODE:-}"
PANEL_DOMAIN="${PANEL_DOMAIN:-}"
LE_EMAIL="${LE_EMAIL:-}"

# ----- pretty -----
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; BOLD='\033[1m'; NC='\033[0m'

ts()    { date '+%Y-%m-%d %H:%M:%S'; }
log()   { printf "${GREEN}[%s] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
warn()  { printf "${YELLOW}[%s] WARN: %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
err()   { printf "${RED}[%s] ERROR: %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE" >&2; }
die()   { err "$*"; exit 1; }

on_error() {
    local rc=$?
    err "Installation failed at line $1 (exit $rc). See $LOG_FILE for details."
    exit "$rc"
}
trap 'on_error $LINENO' ERR

# read из терминала, даже если stdin = process substitution (bash <(curl ...))
ask() {
    # ask "prompt" varname [default]
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
    DEBIAN_FRONTEND=noninteractive apt-get update -qq
    log "Installing prerequisites: git, curl, openssl, ufw, wireguard-tools, iptables, certbot..."
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
        ca-certificates curl gnupg lsb-release git openssl ufw \
        wireguard-tools iptables jq certbot >>"$LOG_FILE" 2>&1
}

install_docker() {
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        log "Docker already installed: $(docker --version | head -n1)"
        return
    fi
    log "Installing Docker Engine + Compose plugin..."
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL "https://download.docker.com/linux/${OS_ID}/gpg" \
        | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${OS_ID} $(lsb_release -cs) stable" \
        > /etc/apt/sources.list.d/docker.list
    DEBIAN_FRONTEND=noninteractive apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
        docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin \
        >>"$LOG_FILE" 2>&1
    systemctl enable --now docker >/dev/null 2>&1 || true
    log "Docker installed: $(docker --version | head -n1)"
}

clone_repo() {
    if [ -d "$INSTALL_DIR/.git" ]; then
        log "Repo already present at $INSTALL_DIR — pulling latest"
        git -C "$INSTALL_DIR" fetch --quiet origin "$REPO_BRANCH"
        git -C "$INSTALL_DIR" reset --hard "origin/$REPO_BRANCH" >>"$LOG_FILE" 2>&1
    else
        log "Cloning $REPO_URL ($REPO_BRANCH) -> $INSTALL_DIR"
        rm -rf "$INSTALL_DIR"
        git clone --depth=1 --branch "$REPO_BRANCH" "$REPO_URL" "$INSTALL_DIR" >>"$LOG_FILE" 2>&1
    fi
    mkdir -p "$INSTALL_DIR/runtime/tls" "$INSTALL_DIR/runtime/acme-www"
}

random_pass() {
    LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 16
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
    SECRETBOX_KEY="$(openssl rand -hex 16)"
    OBFS_PSK="$(openssl rand -hex 32)"
    BOOTSTRAP_ADMIN_PASSWORD="$(random_pass)"
    cat > "$env_file" <<EOF
JWT_SECRET=$JWT_SECRET
SECRETBOX_KEY=$SECRETBOX_KEY
OBFS_PSK=$OBFS_PSK
BOOTSTRAP_ADMIN_EMAIL=$ADMIN_EMAIL
BOOTSTRAP_ADMIN_PASSWORD=$BOOTSTRAP_ADMIN_PASSWORD
PANEL_HTTP_PORT=$PANEL_HTTP_PORT
PANEL_HTTPS_PORT=$PANEL_HTTPS_PORT
WG_PORT=$WG_PORT
OBFS_PORT=$OBFS_PORT
LOG_LEVEL=info
AGENT_INSECURE_TLS=true
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
    ip="$(curl -fsS -4 https://ifconfig.io 2>/dev/null || true)"
    [ -n "$ip" ] || ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
    [ -n "$ip" ] || ip="SERVER_IP"
    echo "$ip"
}

# ===== TLS configuration =====

choose_tls_mode() {
    if [ -n "$TLS_MODE" ]; then
        log "TLS_MODE=$TLS_MODE (from env)"
        return
    fi
    cat <<BANNER

${BOLD}Choose TLS mode for the panel:${NC}
  ${BOLD}1)${NC} ${GREEN}Self-signed${NC} certificate by IP — works immediately,
                  но браузер будет показывать предупреждение.
  ${BOLD}2)${NC} ${GREEN}Let's Encrypt${NC} certificate by domain — нужен валидный домен,
                  указывающий A-записью на этот сервер; cert и автообновление
                  настраиваются автоматически.
  ${BOLD}3)${NC} HTTP only (без TLS) — небезопасно, только для тестов.

BANNER
    local choice
    ask "Select 1/2/3" choice "1"
    case "$choice" in
        1|"") TLS_MODE=selfsigned ;;
        2)    TLS_MODE=letsencrypt ;;
        3)    TLS_MODE=none ;;
        *)    die "Invalid choice: $choice" ;;
    esac
    log "TLS_MODE=$TLS_MODE"
}

ask_domain_if_needed() {
    [ "$TLS_MODE" = "letsencrypt" ] || return 0
    if [ -z "$PANEL_DOMAIN" ]; then
        ask "Enter panel domain (e.g. vpn.example.com)" PANEL_DOMAIN
        [ -n "$PANEL_DOMAIN" ] || die "Domain is required for Let's Encrypt"
    fi
    if [ -z "$LE_EMAIL" ]; then
        local default="$ADMIN_EMAIL"
        [ "$default" = "admin@local" ] && default=""
        ask "Email for Let's Encrypt notifications (recommended)" LE_EMAIL "$default"
    fi
    log "Panel domain: $PANEL_DOMAIN"
}

# Резолвится ли домен в IP сервера? Предупредим, но не падаем — DNS может быть
# свежим, а Let's Encrypt всё равно сделает A-проверку.
check_dns() {
    [ "$TLS_MODE" = "letsencrypt" ] || return 0
    local server_ip; server_ip="$(public_ip)"
    local resolved
    resolved="$(getent ahostsv4 "$PANEL_DOMAIN" 2>/dev/null | awk 'NR==1{print $1}' || true)"
    if [ -z "$resolved" ]; then
        warn "Could not resolve $PANEL_DOMAIN — make sure A-record points to $server_ip"
    elif [ "$resolved" != "$server_ip" ]; then
        warn "$PANEL_DOMAIN resolves to $resolved, but this server's public IP is $server_ip"
        warn "Let's Encrypt will fail unless A-record points to this server."
    else
        log "$PANEL_DOMAIN -> $resolved (matches server IP)"
    fi
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
}

issue_letsencrypt() {
    local domain="$PANEL_DOMAIN"
    local tls_dir="$INSTALL_DIR/runtime/tls"
    local le_dir="/etc/letsencrypt/live/$domain"
    mkdir -p "$tls_dir"

    if [ -f "$le_dir/fullchain.pem" ] && [ -f "$le_dir/privkey.pem" ]; then
        log "Let's Encrypt cert for $domain already present — skipping issuance"
    else
        log "Stopping anything bound to :80 (for ACME http-01)..."
        # На случай если nginx или panel уже слушают порт 80 — стопаем frontend
        ( cd "$INSTALL_DIR" && docker compose stop frontend >/dev/null 2>&1 || true )
        # Параллельно стопаем системный nginx/apache, если есть
        systemctl stop nginx 2>/dev/null || true
        systemctl stop apache2 2>/dev/null || true

        local email_args=()
        if [ -n "$LE_EMAIL" ]; then
            email_args=(--email "$LE_EMAIL" --no-eff-email)
        else
            email_args=(--register-unsafely-without-email)
        fi

        log "Issuing Let's Encrypt certificate for $domain..."
        certbot certonly --standalone --non-interactive --agree-tos \
            "${email_args[@]}" \
            -d "$domain" \
            --preferred-challenges http \
            >>"$LOG_FILE" 2>&1 \
        || die "certbot failed — see $LOG_FILE. Common causes: A-record не указывает на этот сервер, порт 80 закрыт фаерволом."
    fi

    log "Copying certificate into runtime/tls..."
    cp -L "$le_dir/fullchain.pem" "$tls_dir/fullchain.pem"
    cp -L "$le_dir/privkey.pem"   "$tls_dir/privkey.pem"
    chmod 600 "$tls_dir/privkey.pem"
}

write_runtime_nginx_conf() {
    local out="$INSTALL_DIR/runtime/nginx.conf"
    if [ "$TLS_MODE" = "none" ]; then
        log "Writing HTTP-only nginx config"
        cp "$INSTALL_DIR/frontend/nginx.http.conf" "$out"
        return
    fi
    log "Writing HTTPS nginx config (server_name=$PANEL_DOMAIN)"
    sed "s|__SERVER_NAME__|${PANEL_DOMAIN}|g" \
        "$INSTALL_DIR/frontend/nginx.https.conf.tpl" > "$out"
}

configure_tls() {
    choose_tls_mode
    ask_domain_if_needed
    check_dns

    case "$TLS_MODE" in
        selfsigned)  generate_selfsigned ;;
        letsencrypt) issue_letsencrypt ;;
        none)        : ;;
        *) die "Unknown TLS_MODE: $TLS_MODE" ;;
    esac

    write_runtime_nginx_conf

    env_set "TLS_MODE"    "$TLS_MODE"
    env_set "PANEL_DOMAIN" "$PANEL_DOMAIN"
    env_set "LE_EMAIL"    "${LE_EMAIL:-}"
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
    systemctl enable void-wg.service >/dev/null 2>&1 || true
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
    # Симлинк, чтобы обновления через update.sh подхватывались автоматически.
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
    systemctl enable --now void-wg-renew.timer >>"$LOG_FILE" 2>&1
}

start_stack() {
    log "Building and starting containers (first run can take a few minutes)..."
    cd "$INSTALL_DIR"
    docker compose pull --quiet 2>>"$LOG_FILE" || true
    docker compose up -d --build --remove-orphans >>"$LOG_FILE" 2>&1

    log "Waiting for API to be ready..."
    local probe_url
    if [ "$TLS_MODE" = "none" ]; then
        probe_url="http://127.0.0.1:${PANEL_HTTP_PORT}/healthz"
    else
        probe_url="https://127.0.0.1:${PANEL_HTTPS_PORT}/healthz"
    fi
    local ok=0
    for _ in $(seq 1 60); do
        if curl -fsSk "$probe_url" >/dev/null 2>&1; then
            ok=1; break
        fi
        sleep 2
    done
    if [ "$ok" -ne 1 ]; then
        warn "API did not become healthy within 120s. Inspect: docker compose -f $INSTALL_DIR/docker-compose.yml logs"
    else
        log "API is healthy ($probe_url)"
    fi
}

print_summary() {
    local url
    case "$TLS_MODE" in
        letsencrypt) url="https://${PANEL_DOMAIN}" ;;
        selfsigned)  url="https://$(public_ip)" ;;
        none)        url="http://$(public_ip):${PANEL_HTTP_PORT}" ;;
    esac
    [ "$PANEL_HTTPS_PORT" = "443" ] || [ "$TLS_MODE" = "none" ] || url="$url:$PANEL_HTTPS_PORT"

    cat <<EOF

${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}
${BOLD}${GREEN}  Installation complete!${NC}
${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}

  Panel:    ${url}
  TLS mode: ${TLS_MODE}
  Login:    ${ADMIN_EMAIL}
  Password: ${BOOTSTRAP_ADMIN_PASSWORD}

  Files:    ${INSTALL_DIR}
  .env:     ${INSTALL_DIR}/.env  (mode 600 — back it up!)
  TLS dir:  ${INSTALL_DIR}/runtime/tls
  Logs:     docker compose -f ${INSTALL_DIR}/docker-compose.yml logs -f
  Update:   sudo bash ${INSTALL_DIR}/scripts/update.sh
  Renew:    sudo bash ${INSTALL_DIR}/scripts/renew-cert.sh   (Let's Encrypt only)
  Remove:   sudo bash ${INSTALL_DIR}/scripts/uninstall.sh

  ${BOLD}Manage panel:${NC}  sudo v-wg            (interactive menu)

  systemd:  systemctl status void-wg
$( [ "$TLS_MODE" = "letsencrypt" ] && printf '  TLS timer: systemctl status void-wg-renew.timer\n' )

${YELLOW}Save these credentials — they are also stored in ${INSTALL_DIR}/.env${NC}
$( [ "$TLS_MODE" = "selfsigned" ] && printf "${YELLOW}Self-signed cert: браузер покажет предупреждение — это нормально для IP-режима.${NC}\n" )

${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}
EOF
}

main() {
    require_root
    mkdir -p "$(dirname "$LOG_FILE")"
    : > "$LOG_FILE"
    log "void-wg installer starting..."

    detect_os
    install_apt_packages
    install_docker
    clone_repo
    ensure_env_file
    configure_tls
    configure_firewall
    install_systemd_unit
    install_renew_timer
    install_cli_wrapper
    start_stack
    print_summary
}

main "$@"

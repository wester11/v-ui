#!/usr/bin/env bash
# void-wg management TUI.
#
# Установка: install.sh кладёт этот файл в /usr/local/bin/v-wg.
# Запуск:    v-wg                   — интерактивное меню
#            v-wg <num>              — выполнить пункт меню напрямую
#            v-wg status|start|stop|restart|logs|update|renew  — алиасы
#
# Цель: повторить UX `x-ui` (3x-ui) для void-wg.

set -Eeuo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/void-wg}"
COMPOSE="docker compose -f $INSTALL_DIR/docker-compose.yml"

# ----- pretty -----
B='\033[1m'; R='\033[0;31m'; G='\033[0;32m'; Y='\033[1;33m'; C='\033[0;36m'; D='\033[2m'; N='\033[0m'

die()  { printf "${R}%s${N}\n" "$*" >&2; exit 1; }
info() { printf "${G}%s${N}\n" "$*"; }
warn() { printf "${Y}%s${N}\n" "$*"; }
hr()   { printf "${D}%s${N}\n" "────────────────────────────────────────────────"; }
press_enter() {
    printf "\n${D}Press Enter to continue...${N}"
    if [ -e /dev/tty ]; then read -r _ < /dev/tty || true; else read -r _ || true; fi
}

# read из терминала, даже если stdin = process substitution
ask() {
    # ask "prompt" varname [default]
    local prompt="$1" varname="$2" default="${3:-}" reply
    [ -n "$default" ] && prompt="$prompt [$default]"
    if [ -e /dev/tty ]; then
        read -r -p "$prompt: " reply < /dev/tty || reply=""
    else
        read -r -p "$prompt: " reply || reply=""
    fi
    [ -z "$reply" ] && reply="$default"
    printf -v "$varname" '%s' "$reply"
}

ask_yes() {
    local prompt="$1" reply
    if [ -e /dev/tty ]; then
        read -r -p "$prompt [y/N]: " reply < /dev/tty || reply=""
    else
        read -r -p "$prompt [y/N]: " reply || reply=""
    fi
    [[ "$reply" =~ ^[yY](es)?$ ]]
}

require_root() {
    [ "$(id -u)" -eq 0 ] || die "Run as root: sudo v-wg"
}

require_install() {
    [ -d "$INSTALL_DIR/.git" ] || die "void-wg not installed at $INSTALL_DIR. Run install.sh first."
    [ -f "$INSTALL_DIR/.env" ] || die "$INSTALL_DIR/.env missing"
}

load_env() {
    # shellcheck disable=SC1090
    set -a; . "$INSTALL_DIR/.env"; set +a
}

env_get() {
    grep -E "^$1=" "$INSTALL_DIR/.env" 2>/dev/null | head -n1 | cut -d= -f2- | tr -d '"'
}

env_set() {
    local key="$1" value="$2" file="$INSTALL_DIR/.env"
    if grep -q "^${key}=" "$file" 2>/dev/null; then
        sed -i "s|^${key}=.*|${key}=${value}|" "$file"
    else
        printf '%s=%s\n' "$key" "$value" >> "$file"
    fi
}

random_pass() { LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 16; }

# ----- status helpers -----

panel_state() {
    local out
    out="$($COMPOSE ps --format '{{.Service}} {{.State}}' 2>/dev/null || true)"
    if [ -z "$out" ]; then printf "%bnot deployed%b" "$Y" "$N"; return; fi
    local up=0 total=0
    while IFS= read -r line; do
        [ -z "$line" ] && continue
        total=$((total + 1))
        case "$line" in *running*|*Up*) up=$((up + 1)) ;; esac
    done <<< "$out"
    if [ "$up" -eq 0 ]; then printf "%bStopped%b"   "$R" "$N"
    elif [ "$up" -lt "$total" ]; then printf "%bDegraded (%d/%d)%b" "$Y" "$up" "$total" "$N"
    else printf "%bRunning (%d/%d)%b" "$G" "$up" "$total" "$N"
    fi
}

autostart_state() {
    if systemctl is-enabled void-wg.service >/dev/null 2>&1; then
        printf "%bEnabled%b" "$G" "$N"
    else
        printf "%bDisabled%b" "$Y" "$N"
    fi
}

tls_state() {
    local mode domain
    mode="$(env_get TLS_MODE)"; mode="${mode:-none}"
    domain="$(env_get PANEL_DOMAIN)"
    case "$mode" in
        letsencrypt) printf "%bLet's Encrypt%b (%s)" "$G" "$N" "$domain" ;;
        selfsigned)  printf "%bself-signed%b (%s)"   "$Y" "$N" "${domain:-IP}" ;;
        none)        printf "%bHTTP-only%b"          "$R" "$N" ;;
        *)           printf "%bunknown%b"            "$R" "$N" ;;
    esac
}

# ----- menu actions -----

action_install() {
    info "Running install.sh (idempotent)..."
    bash "$INSTALL_DIR/scripts/install.sh"
}

action_update() {
    info "Running update.sh..."
    bash "$INSTALL_DIR/scripts/update.sh"
}

action_uninstall() {
    bash "$INSTALL_DIR/scripts/uninstall.sh"
}

action_reset_password() {
    require_install; load_env
    local email="$BOOTSTRAP_ADMIN_EMAIL"
    cat <<EOF

Reset password for admin user: ${B}${email}${N}

This will:
  1) DELETE the existing admin row (peers owned by admin will be cascade-deleted!).
  2) Write a new random password into .env.
  3) Restart api — bootstrap will recreate the admin with the new password.

EOF
    if ! ask_yes "Proceed?"; then info "Aborted."; return; fi

    local new_pass; new_pass="$(random_pass)"
    info "Generated new password: ${B}${new_pass}${N}"

    info "Removing existing admin from database..."
    $COMPOSE exec -T postgres psql -U voidwg -d voidwg -v ON_ERROR_STOP=1 \
        -c "DELETE FROM users WHERE email = '${email}';" >/dev/null

    env_set BOOTSTRAP_ADMIN_PASSWORD "$new_pass"
    chmod 600 "$INSTALL_DIR/.env"

    info "Restarting api so bootstrap recreates the admin..."
    $COMPOSE up -d api >/dev/null
    $COMPOSE restart api >/dev/null

    sleep 3
    info "Done."
    cat <<EOF

  Login:    ${email}
  Password: ${B}${new_pass}${N}

EOF
}

action_view_credentials() {
    require_install; load_env
    cat <<EOF

  Login:    ${BOOTSTRAP_ADMIN_EMAIL:-?}
  Password: ${B}${BOOTSTRAP_ADMIN_PASSWORD:-?}${N}

  ${D}(stored in $INSTALL_DIR/.env, mode 600)${N}

EOF
}

action_view_settings() {
    require_install; load_env
    local mode domain le_email http_port https_port wg_port obfs_port
    mode="$(env_get TLS_MODE)";          mode="${mode:-none}"
    domain="$(env_get PANEL_DOMAIN)"
    le_email="$(env_get LE_EMAIL)"
    http_port="$(env_get PANEL_HTTP_PORT)";  http_port="${http_port:-80}"
    https_port="$(env_get PANEL_HTTPS_PORT)"; https_port="${https_port:-443}"
    wg_port="$(env_get WG_PORT)";        wg_port="${wg_port:-51820}"
    obfs_port="$(env_get OBFS_PORT)";    obfs_port="${obfs_port:-51821}"

    local panel_url
    case "$mode" in
        letsencrypt) panel_url="https://${domain}" ;;
        selfsigned)  panel_url="https://${domain:-server-ip}" ;;
        none)        panel_url="http://server-ip:${http_port}" ;;
    esac

    cat <<EOF

  ${B}Panel${N}
    URL:           $panel_url
    HTTP port:     $http_port
    HTTPS port:    $https_port
    Admin email:   ${BOOTSTRAP_ADMIN_EMAIL:-?}

  ${B}TLS${N}
    Mode:          $mode
    Domain:        ${domain:-—}
    LE email:      ${le_email:-—}

  ${B}WireGuard${N}
    Listen port:   $wg_port/udp
    Obfs port:     $obfs_port/udp

  ${B}Files${N}
    Install dir:   $INSTALL_DIR
    Env file:      $INSTALL_DIR/.env  (mode 600)
    TLS dir:       $INSTALL_DIR/runtime/tls
    Logs:          $COMPOSE logs -f

EOF
}

action_change_ports() {
    require_install; load_env
    local cur_http cur_https cur_wg cur_obfs new
    cur_http="$(env_get PANEL_HTTP_PORT)";  cur_http="${cur_http:-80}"
    cur_https="$(env_get PANEL_HTTPS_PORT)"; cur_https="${cur_https:-443}"
    cur_wg="$(env_get WG_PORT)";        cur_wg="${cur_wg:-51820}"
    cur_obfs="$(env_get OBFS_PORT)";    cur_obfs="${cur_obfs:-51821}"

    info "Current ports: HTTP=$cur_http, HTTPS=$cur_https, WG=$cur_wg/udp, Obfs=$cur_obfs/udp"
    info "Press Enter to keep current value."

    ask "HTTP port"   new "$cur_http";  [ "$new" != "$cur_http" ]  && env_set PANEL_HTTP_PORT  "$new"
    ask "HTTPS port"  new "$cur_https"; [ "$new" != "$cur_https" ] && env_set PANEL_HTTPS_PORT "$new"
    ask "WG port"     new "$cur_wg";    [ "$new" != "$cur_wg" ]    && env_set WG_PORT        "$new"
    ask "Obfs port"   new "$cur_obfs";  [ "$new" != "$cur_obfs" ]  && env_set OBFS_PORT      "$new"

    if ask_yes "Restart stack and update firewall?"; then
        if command -v ufw >/dev/null 2>&1; then
            for p in "$cur_http/tcp" "$cur_https/tcp" "$cur_wg/udp" "$cur_obfs/udp"; do
                ufw delete allow "$p" >/dev/null 2>&1 || true
            done
            load_env
            ufw allow "${PANEL_HTTP_PORT}/tcp"  >/dev/null
            ufw allow "${PANEL_HTTPS_PORT}/tcp" >/dev/null
            ufw allow "${WG_PORT}/udp"   >/dev/null
            ufw allow "${OBFS_PORT}/udp" >/dev/null
            info "ufw updated."
        fi
        $COMPOSE up -d --force-recreate
        info "Stack restarted with new ports."
    fi
}

action_start()   { require_install; $COMPOSE up -d --remove-orphans; info "Started."; }
action_stop()    { require_install; $COMPOSE stop;                   info "Stopped."; }
action_restart() { require_install; $COMPOSE restart;                info "Restarted."; }

action_status() {
    require_install
    info "=== docker compose ps ==="
    $COMPOSE ps
    echo
    info "=== systemd ==="
    systemctl status void-wg.service --no-pager 2>/dev/null | head -n 8 || echo "(not enabled)"
    echo
    if systemctl list-timers void-wg-renew.timer >/dev/null 2>&1; then
        info "=== TLS renewal timer ==="
        systemctl list-timers void-wg-renew.timer --no-pager
    fi
}

action_logs() {
    require_install
    info "Streaming logs (Ctrl+C to exit)..."
    $COMPOSE logs -f --tail=100
}

action_autostart_enable() {
    require_install
    systemctl enable void-wg.service
    info "Autostart enabled."
}

action_autostart_disable() {
    require_install
    systemctl disable void-wg.service
    info "Autostart disabled (containers keep running)."
}

action_renew_now() {
    require_install
    info "Running renew-cert.sh..."
    bash "$INSTALL_DIR/scripts/renew-cert.sh"
}

action_switch_tls_mode() {
    require_install; load_env
    cat <<EOF

Current TLS mode: $(tls_state)

Choose new mode:
  ${B}1)${N} Self-signed by IP
  ${B}2)${N} Let's Encrypt by domain
  ${B}3)${N} HTTP-only (no TLS)

EOF
    local choice; ask "Select 1/2/3" choice
    case "$choice" in
        1) info "Switching to self-signed mode..."
           sudo TLS_MODE=selfsigned bash "$INSTALL_DIR/scripts/install.sh" ;;
        2) local d e
           ask "Domain (A-record must point here)" d
           ask "Email for Let's Encrypt (optional)" e
           [ -n "$d" ] || { warn "Domain is required."; return; }
           info "Switching to Let's Encrypt mode..."
           sudo TLS_MODE=letsencrypt PANEL_DOMAIN="$d" LE_EMAIL="$e" \
                bash "$INSTALL_DIR/scripts/install.sh" ;;
        3) info "Switching to HTTP-only mode..."
           sudo TLS_MODE=none bash "$INSTALL_DIR/scripts/install.sh" ;;
        *) warn "Invalid choice."; return ;;
    esac
}

action_firewall() {
    if ! command -v ufw >/dev/null 2>&1; then
        warn "ufw not installed — skipping."; return
    fi
    info "Current ufw rules:"
    ufw status numbered || true
    echo
    cat <<EOF
Actions:
  ${B}1)${N} Allow port (TCP)
  ${B}2)${N} Allow port (UDP)
  ${B}3)${N} Delete rule by number
  ${B}4)${N} Disable ufw
  ${B}5)${N} Enable ufw
  ${B}0)${N} Back
EOF
    local choice; ask "Select" choice "0"
    case "$choice" in
        1) local p; ask "Port" p; [ -n "$p" ] && ufw allow "${p}/tcp" ;;
        2) local p; ask "Port" p; [ -n "$p" ] && ufw allow "${p}/udp" ;;
        3) local n; ask "Rule number" n; [ -n "$n" ] && ufw --force delete "$n" ;;
        4) ufw --force disable ;;
        5) ufw --force enable ;;
    esac
}

action_enable_bbr() {
    require_root
    if sysctl net.ipv4.tcp_congestion_control 2>/dev/null | grep -q 'bbr'; then
        info "BBR is already enabled."
        return
    fi
    info "Enabling BBR..."
    cat > /etc/sysctl.d/99-void-wg-bbr.conf <<EOF
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
EOF
    sysctl --system >/dev/null
    if sysctl net.ipv4.tcp_congestion_control 2>/dev/null | grep -q 'bbr'; then
        info "BBR enabled."
    else
        warn "BBR not active — kernel may not support it. Check: lsmod | grep bbr"
    fi
}

# ----- menu -----

show_header() {
    local os="${PRETTY_NAME:-$(uname -s)}"
    [ -f /etc/os-release ] && . /etc/os-release && os="$PRETTY_NAME"
    clear
    printf "\n"
    printf "${C}The OS release is: ${B}%s${N}\n" "$os"
}

show_menu() {
    printf "${C}╔──────────────────────────────────────────────────╗${N}\n"
    printf "${C}│${N}   ${B}void-wg Panel Management Script${N}                ${C}│${N}\n"
    printf "${C}│${N}   0. Exit                                        ${C}│${N}\n"
    printf "${C}│──────────────────────────────────────────────────│${N}\n"
    printf "${C}│${N}   1. Install / Reinstall                         ${C}│${N}\n"
    printf "${C}│${N}   2. Update                                      ${C}│${N}\n"
    printf "${C}│${N}   3. Uninstall                                   ${C}│${N}\n"
    printf "${C}│──────────────────────────────────────────────────│${N}\n"
    printf "${C}│${N}   4. Reset admin password                        ${C}│${N}\n"
    printf "${C}│${N}   5. View current credentials                    ${C}│${N}\n"
    printf "${C}│${N}   6. View settings                               ${C}│${N}\n"
    printf "${C}│${N}   7. Change panel ports                          ${C}│${N}\n"
    printf "${C}│──────────────────────────────────────────────────│${N}\n"
    printf "${C}│${N}   8. Start                                       ${C}│${N}\n"
    printf "${C}│${N}   9. Stop                                        ${C}│${N}\n"
    printf "${C}│${N}  10. Restart                                     ${C}│${N}\n"
    printf "${C}│${N}  11. Status                                      ${C}│${N}\n"
    printf "${C}│${N}  12. Live logs                                   ${C}│${N}\n"
    printf "${C}│──────────────────────────────────────────────────│${N}\n"
    printf "${C}│${N}  13. Enable autostart                            ${C}│${N}\n"
    printf "${C}│${N}  14. Disable autostart                           ${C}│${N}\n"
    printf "${C}│──────────────────────────────────────────────────│${N}\n"
    printf "${C}│${N}  15. TLS: renew certificate now                  ${C}│${N}\n"
    printf "${C}│${N}  16. TLS: switch mode (IP / domain / none)       ${C}│${N}\n"
    printf "${C}│${N}  17. Firewall management (ufw)                   ${C}│${N}\n"
    printf "${C}│──────────────────────────────────────────────────│${N}\n"
    printf "${C}│${N}  18. Enable BBR                                  ${C}│${N}\n"
    printf "${C}╚──────────────────────────────────────────────────╝${N}\n"
    if [ -f "$INSTALL_DIR/.env" ]; then
        printf "Panel state: %b\n"   "$(panel_state)"
        printf "Autostart:   %b\n"   "$(autostart_state)"
        printf "TLS mode:    %b\n"   "$(tls_state)"
    else
        printf "${Y}Panel not installed at %s${N}\n" "$INSTALL_DIR"
    fi
}

dispatch() {
    case "$1" in
        0|exit|q)        exit 0 ;;
        1|install)       action_install ;;
        2|update)        action_update ;;
        3|uninstall)     action_uninstall ;;
        4|reset-pass|resetpw) action_reset_password ;;
        5|creds)         action_view_credentials ;;
        6|settings|view) action_view_settings ;;
        7|ports)         action_change_ports ;;
        8|start)         action_start ;;
        9|stop)          action_stop ;;
        10|restart)      action_restart ;;
        11|status)       action_status ;;
        12|logs)         action_logs ;;
        13|enable-autostart) action_autostart_enable ;;
        14|disable-autostart) action_autostart_disable ;;
        15|renew)        action_renew_now ;;
        16|tls)          action_switch_tls_mode ;;
        17|firewall|ufw) action_firewall ;;
        18|bbr)          action_enable_bbr ;;
        *)               warn "Unknown selection: $1" ;;
    esac
}

main() {
    require_root

    # Алиас-режим: v-wg <subcommand> или v-wg <number>
    if [ "$#" -ge 1 ]; then
        dispatch "$1"
        exit 0
    fi

    while true; do
        show_header
        show_menu
        local choice
        ask "Please enter your selection [0-18]" choice "0"
        echo
        if [ "$choice" = "0" ] || [ -z "$choice" ]; then
            info "Bye."
            exit 0
        fi
        dispatch "$choice"
        press_enter
    done
}

main "$@"

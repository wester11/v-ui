#!/usr/bin/env bash
# Обновление TLS-сертификата void-wg.
#
# Запускается:
#   * вручную:   sudo bash /opt/void-wg/scripts/renew-cert.sh
#   * по таймеру: systemd-timer void-wg-renew.timer (дважды в сутки)
#
# Поведение:
#   - TLS_MODE=letsencrypt: вызывает certbot renew. Если cert обновился —
#     копирует в runtime/tls и делает nginx reload без рестарта frontend.
#     Если certbot не может выполнить ACME challenge при работающем nginx,
#     он сам остановит/запустит frontend через pre/post-hook.
#   - TLS_MODE=selfsigned: пересоздаёт self-signed cert (если осталось < 30 дней)
#     и перезагружает nginx.
#   - TLS_MODE=none: ничего не делает.
#
# Идемпотентен. Логи: /var/log/void-wg-renew.log

set -Eeuo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/void-wg}"
LOG_FILE="${LOG_FILE:-/var/log/void-wg-renew.log}"
COMPOSE="docker compose -f $INSTALL_DIR/docker-compose.yml"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
ts()   { date '+%Y-%m-%d %H:%M:%S'; }
log()  { printf "${GREEN}[%s] %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
warn() { printf "${YELLOW}[%s] WARN: %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE"; }
err()  { printf "${RED}[%s] ERROR: %s${NC}\n" "$(ts)" "$*" | tee -a "$LOG_FILE" >&2; }
die()  { err "$*"; exit 1; }

[ "$(id -u)" -eq 0 ] || die "Run as root"
[ -f "$INSTALL_DIR/.env" ] || die "$INSTALL_DIR/.env not found"

mkdir -p "$(dirname "$LOG_FILE")"

# shellcheck disable=SC1090
set -a; . "$INSTALL_DIR/.env"; set +a

TLS_MODE="${TLS_MODE:-none}"
PANEL_DOMAIN="${PANEL_DOMAIN:-}"
TLS_DIR="$INSTALL_DIR/runtime/tls"

reload_nginx() {
    if $COMPOSE ps frontend 2>/dev/null | grep -q 'Up\|running'; then
        log "Reloading nginx in frontend container..."
        $COMPOSE exec -T frontend nginx -s reload >>"$LOG_FILE" 2>&1 || \
            $COMPOSE restart frontend >>"$LOG_FILE" 2>&1
    else
        log "frontend container not running — starting..."
        $COMPOSE up -d frontend >>"$LOG_FILE" 2>&1 || true
    fi
}

renew_letsencrypt() {
    [ -n "$PANEL_DOMAIN" ] || die "PANEL_DOMAIN missing in .env"
    command -v certbot >/dev/null 2>&1 || die "certbot not installed (apt install certbot)"

    local le_dir="/etc/letsencrypt/live/$PANEL_DOMAIN"
    if [ ! -d "$le_dir" ]; then
        die "Let's Encrypt cert directory $le_dir missing — re-run install.sh"
    fi

    log "Running certbot renew (will only act if cert expires in < 30 days)..."
    # --pre-hook/--post-hook СРАБАТЫВАЮТ только когда certbot реально пытается
    # обновить сертификат (no-op runs не трогают frontend).
    certbot renew --quiet --no-random-sleep-on-renew \
        --pre-hook  "$COMPOSE stop frontend" \
        --post-hook "$COMPOSE start frontend" \
        >>"$LOG_FILE" 2>&1 \
    || { warn "certbot renew exited non-zero (см. $LOG_FILE)"; return 1; }

    # certbot обновляет /etc/letsencrypt/live/<domain>/* — копируем в runtime/tls.
    # Сравниваем mtime, чтобы понять, было ли обновление.
    local src="$le_dir/fullchain.pem"
    local dst="$TLS_DIR/fullchain.pem"
    if [ -f "$src" ] && { [ ! -f "$dst" ] || [ "$src" -nt "$dst" ]; }; then
        log "Cert was renewed — copying to $TLS_DIR"
        cp -L "$src"                "$TLS_DIR/fullchain.pem"
        cp -L "$le_dir/privkey.pem" "$TLS_DIR/privkey.pem"
        chmod 600 "$TLS_DIR/privkey.pem"
        reload_nginx
        log "Renewal complete: $PANEL_DOMAIN"
    else
        log "Cert is fresh — nothing to do"
    fi
}

renew_selfsigned() {
    local cert="$TLS_DIR/fullchain.pem"
    if [ ! -f "$cert" ]; then
        warn "No self-signed cert found at $cert — generating now"
    else
        # сколько дней осталось до истечения
        local end days_left now end_ts
        end="$(openssl x509 -in "$cert" -enddate -noout 2>/dev/null | cut -d= -f2)"
        end_ts="$(date -d "$end" +%s 2>/dev/null || echo 0)"
        now="$(date +%s)"
        days_left=$(( (end_ts - now) / 86400 ))
        if [ "$days_left" -gt 30 ]; then
            log "Self-signed cert valid for $days_left more days — skipping"
            return
        fi
        log "Self-signed cert expires in $days_left days — regenerating"
    fi

    local ip
    ip="$(curl -fsS -4 https://ifconfig.io 2>/dev/null \
          || hostname -I 2>/dev/null | awk '{print $1}')"
    [ -n "$ip" ] || die "Could not determine server IP"

    mkdir -p "$TLS_DIR"
    openssl req -x509 -nodes -newkey rsa:2048 -days 3650 \
        -keyout "$TLS_DIR/privkey.pem" \
        -out    "$TLS_DIR/fullchain.pem" \
        -subj   "/CN=$ip" \
        -addext "subjectAltName=IP:$ip,DNS:localhost" \
        >>"$LOG_FILE" 2>&1
    chmod 600 "$TLS_DIR/privkey.pem"
    reload_nginx
    log "Self-signed cert regenerated for $ip"
}

case "$TLS_MODE" in
    letsencrypt) renew_letsencrypt ;;
    selfsigned)  renew_selfsigned ;;
    none)        log "TLS_MODE=none — nothing to renew" ;;
    *)           die "Unknown TLS_MODE: $TLS_MODE" ;;
esac

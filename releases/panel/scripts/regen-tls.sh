#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════
#  void-wg  — пересоздать TLS-сертификат + nginx конфиг
#
#  Запускать на VPS:
#    sudo bash /opt/void-wg/scripts/regen-tls.sh
#
#  Исправляет: самоподписанный сертификат без IP SAN
#  Результат : перезапущенный frontend с рабочим HTTPS
# ═══════════════════════════════════════════════════════════
set -Eeuo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/void-wg}"
cd "$INSTALL_DIR"

G='\033[0;32m'; Y='\033[1;33m'; R='\033[0;31m'; N='\033[0m'
log()  { printf "${G}✓ %s${N}\n" "$*"; }
info() { printf "  » %s\n" "$*"; }
warn() { printf "${Y}⚠ %s${N}\n" "$*"; }
err()  { printf "${R}✗ %s${N}\n" "$*" >&2; }
die()  { err "$*"; exit 1; }

[[ "$(id -u)" -eq 0 ]] || die "Run as root: sudo bash $0"

# ── Load existing .env ────────────────────────────────────
ENV_FILE="$INSTALL_DIR/.env"
[[ -f "$ENV_FILE" ]] || die ".env not found at $ENV_FILE"

# shellcheck disable=SC1090
set -a; source "$ENV_FILE"; set +a

PUBLIC_BASE_URL="${PUBLIC_BASE_URL:-}"
PANEL_ENTRY_TOKEN="${PANEL_ENTRY_TOKEN:-}"
PANEL_HTTPS_PORT="${PANEL_HTTPS_PORT:-443}"
PANEL_DOMAIN="${PANEL_DOMAIN:-}"

[[ -n "$PUBLIC_BASE_URL" ]] || die "PUBLIC_BASE_URL not set in .env"

# Extract IP/domain from PUBLIC_BASE_URL
PANEL_DOMAIN_RAW="${PUBLIC_BASE_URL#https://}"
PANEL_DOMAIN_RAW="${PANEL_DOMAIN_RAW#http://}"
PANEL_DOMAIN_RAW="${PANEL_DOMAIN_RAW%%/*}"
PANEL_DOMAIN_RAW="${PANEL_DOMAIN_RAW%%:*}"

printf "\n${G}════════════════════════════════${N}\n"
printf "${G} void-wg TLS regen${N}\n"
printf "${G}════════════════════════════════${N}\n\n"

info "Install dir  : $INSTALL_DIR"
info "Domain/IP    : $PANEL_DOMAIN_RAW"
info "HTTPS port   : $PANEL_HTTPS_PORT"
info "Entry token  : ${PANEL_ENTRY_TOKEN:0:8}..."

TLS_DIR="$INSTALL_DIR/runtime/tls"
mkdir -p "$TLS_DIR"

# ── Generate certificate ──────────────────────────────────
# Check if it's an IP address
if [[ "$PANEL_DOMAIN_RAW" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  SAN_TYPE="IP"
  info "Mode         : self-signed (IP SAN)"
else
  SAN_TYPE="DNS"
  info "Mode         : self-signed (DNS SAN)"
fi

SSL_CFG="$(mktemp /tmp/void-ssl-XXXX.cnf)"
cat > "$SSL_CFG" << SSLEOF
[req]
prompt             = no
distinguished_name = dn
x509_extensions    = san_ext

[dn]
CN = $PANEL_DOMAIN_RAW
O  = VoidVPN

[san_ext]
subjectAltName   = ${SAN_TYPE}:${PANEL_DOMAIN_RAW},DNS:localhost
keyUsage         = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth, clientAuth
basicConstraints = CA:FALSE
SSLEOF

info "Generating RSA-4096 certificate (10 years)..."
openssl req -x509 -nodes -newkey rsa:4096 -days 3650 \
    -keyout "$TLS_DIR/privkey.pem" \
    -out    "$TLS_DIR/fullchain.pem" \
    -config "$SSL_CFG" 2>&1 | tail -3
rm -f "$SSL_CFG"
chmod 600 "$TLS_DIR/privkey.pem"
log "Certificate generated"

# Verify SAN
info "Verifying SAN field..."
openssl x509 -in "$TLS_DIR/fullchain.pem" -noout -ext subjectAltName 2>/dev/null \
  && log "SAN verified" \
  || warn "Could not verify SAN (OpenSSL < 1.1.1 — may still work)"

# ── Write nginx HTTPS config ──────────────────────────────
TPL="$INSTALL_DIR/frontend/nginx.https.conf.tpl"
NGINX_CONF="$INSTALL_DIR/runtime/nginx.conf"

if [[ ! -f "$TPL" ]]; then
  die "Template not found: $TPL"
fi

sed \
  -e "s|__SERVER_NAME__|${PANEL_DOMAIN_RAW}|g" \
  -e "s|__HTTPS_PORT__|${PANEL_HTTPS_PORT}|g" \
  -e "s|__PANEL_ENTRY_TOKEN__|${PANEL_ENTRY_TOKEN}|g" \
  "$TPL" > "$NGINX_CONF"
log "nginx config written"

# ── Restart frontend ──────────────────────────────────────
info "Restarting frontend container..."
docker compose restart frontend
log "Frontend restarted"

printf "\n${G}════════════════════════════════${N}\n"
printf "${G} Done! Your panel:${N}\n\n"
printf "  ${G}https://%s:%s/%s${N}\n" "$PANEL_DOMAIN_RAW" "$PANEL_HTTPS_PORT" "$PANEL_ENTRY_TOKEN"
printf "\n${Y}Note: browser will warn about self-signed cert.${N}\n"
printf "${Y}To fix: get a domain + run install.sh with Let's Encrypt mode.${N}\n\n"

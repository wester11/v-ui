#!/usr/bin/env bash
set -Eeuo pipefail

CONTROL_URL=""
NODE_ID=""
SECRET=""
NODE_VERSION="${NODE_VERSION:-latest}"

for arg in "$@"; do
  case "$arg" in
    --control-url=*) CONTROL_URL="${arg#*=}" ;;
    --node-id=*) NODE_ID="${arg#*=}" ;;
    --secret=*) SECRET="${arg#*=}" ;;
  esac
done

if [ "$(id -u)" -ne 0 ]; then
  echo "run as root" >&2
  exit 1
fi

[ -n "$CONTROL_URL" ] || { echo "--control-url is required" >&2; exit 1; }
[ -n "$NODE_ID" ] || { echo "--node-id is required" >&2; exit 1; }
[ -n "$SECRET" ] || { echo "--secret is required" >&2; exit 1; }

export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y -qq ca-certificates curl gnupg lsb-release

if ! command -v docker >/dev/null 2>&1; then
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL "https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg" | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  chmod a+r /etc/apt/keyrings/docker.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list
  apt-get update -qq
  apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  systemctl enable --now docker
fi

mkdir -p /opt/void-node
cat >/opt/void-node/docker-compose.yml <<EOF
services:
  void-node:
    image: void/node:${NODE_VERSION}
    network_mode: host
    restart: always
    environment:
      - CONTROL_URL=${CONTROL_URL}
      - NODE_ID=${NODE_ID}
      - SECRET=${SECRET}
      - HTTP_LISTEN=:7443
EOF

cd /opt/void-node
docker compose pull || true
docker compose up -d --remove-orphans
echo "node agent started"

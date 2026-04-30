# void-wg · node branch

Node agent for WireGuard / AmneziaWG servers. Runs on any VPS, registers with the control panel, and handles peer config management + traffic reporting.

## Quick install

Copy the command from the panel UI (Servers → Add Server) and run it on the node VPS:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/wester11/v-ui/node/scripts/install-node.sh) \
  --control-url https://PANEL_IP \
  --node-id NODE_ID \
  --secret SECRET \
  --token AGENT_TOKEN \
  --protocol amneziawg
```

The script installs Docker, deploys the agent container, and the node appears ONLINE in the panel within ~30 seconds.

## Manual deploy

```bash
cp .env.example .env
# fill in CONTROL_URL, AGENT_TOKEN, NODE_ID, SECRET
nano .env

docker compose up -d
```

## Supported protocols

| Protocol | Port | Notes |
|---|---|---|
| `amneziawg` | 51821 (UDP) | Default. AmneziaWG obfuscation over WireGuard |
| `wireguard` | 51820 (UDP) | Standard WireGuard, no obfuscation |

## Agent API endpoints

The agent exposes an HTTP API on `:7443` (token-protected):

| Endpoint | Method | Description |
|---|---|---|
| `/healthz` | GET | Liveness check |
| `/v1/metrics` | GET | System + WG peer stats |
| `/v1/peers` | POST/DELETE | Add / remove WG peers |
| `/v1/restart` | POST | Restart transport |

## Ports to open

| Port | Protocol | Purpose |
|---|---|---|
| 51821 | UDP | AmneziaWG (obfuscated WireGuard) |
| 51820 | UDP | Plain WireGuard (if protocol=wireguard) |
| 7443 | TCP | Agent API (panel → node) |

## Environment variables

See `.env.example` for all variables and their defaults.

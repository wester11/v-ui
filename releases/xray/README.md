# void-wg · xray branch

Node agent for Xray-Reality servers. Runs the agent + Xray sidecar containers. The control panel pushes a full `config.json` on each config change; the agent writes it atomically and restarts Xray.

## Quick install

Copy the command from the panel UI (Servers → Add Server → Protocol: Xray) and run on the node VPS:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/wester11/v-ui/xray/scripts/install-node.sh) \
  --control-url https://PANEL_IP \
  --node-id NODE_ID \
  --secret SECRET \
  --token AGENT_TOKEN \
  --protocol xray
```

## Manual deploy

```bash
cp .env.example .env
nano .env   # fill in CONTROL_URL, AGENT_TOKEN, NODE_ID, SECRET

mkdir -p runtime/xray
docker compose up -d
```

## How config push works

```
Panel → POST /v1/xray/deploy (full config.json body)
  → Agent writes /etc/xray/config.json atomically
  → Agent runs: docker restart xray
  → Agent probes XRAY_HEALTH TCP port
  → Returns 204 on success
```

## Agent API endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/healthz` | GET | Liveness check |
| `/v1/metrics` | GET | System stats + xray health |
| `/v1/xray/deploy` | POST | Push full config.json |
| `/v1/xray/health` | GET | TCP probe xray inbound port |
| `/v1/restart` | POST | docker restart xray |

## Ports to open

| Port | Protocol | Purpose |
|---|---|---|
| 443 | TCP/UDP | Xray-Reality (HTTPS mimicry) |
| 7443 | TCP | Agent API (panel → node) |

## Environment variables

See `.env.example` for all variables and their defaults.

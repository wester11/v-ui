<div align="center">

# void-wg

**Production-grade VPN management panel for WireGuard, AmneziaWG and Xray**

[![License: MIT](https://img.shields.io/badge/License-MIT-purple.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react)](https://react.dev)
[![Docker](https://img.shields.io/badge/Docker-required-2496ED?logo=docker)](https://docker.com)

[Install](#-quick-install) · [Features](#-features) · [Architecture](#-architecture) · [Clients](#-vpn-clients) · [FAQ](#-faq)

</div>

---

## Overview

void-wg is a self-hosted VPN control plane. You run one **panel** on your management server, then onboard as many **node VPS** machines as you need. Each node runs a lightweight agent that reports health, peer stats and traffic metrics back to the panel in real time.

The panel gives you a dark SaaS-style web UI to:

- Manage VPN nodes (WireGuard / AmneziaWG / Xray-Reality)
- Create and distribute peer configs (`.conf` / `.vwg`)
- Monitor live traffic, peer connections and system health
- Invite users and manage access

---

## ⚡ Quick Install

> **Requirements:** Ubuntu 20.04+ · Docker · Port 443 open · 1 GB RAM minimum

```bash
bash <(curl -Ls https://raw.githubusercontent.com/wester11/v-ui/main/scripts/install.sh)
```

The installer will ask you three questions and then deploy everything with Docker Compose:

1. **Admin email** — your login email
2. **Admin password** — your panel login password
3. **TLS mode** — choose between:
   - `1` — Let's Encrypt (requires a domain pointing to the server)
   - `2` — Self-signed certificate (IP-only, browser will warn once)

After install you'll see your panel URL with a secret entry token:

```
✓ Panel ready: https://1.2.3.4/s3cr3t-t0k3n
```

Nobody can reach the panel without knowing this path — even a direct hit to the IP returns 404.

### Re-install / Update

Run the same command again. The installer detects an existing `.env` and offers:

```
TLS mode: 1-Keep current  2-Domain  3-IP self-signed
```

To pull latest code and rebuild containers only (no config questions):

```bash
cd /opt/void-wg && git pull && docker compose up -d --build
```

---

## ✨ Features

### Panel
- Dark SaaS UI (Remnawave-inspired design)
- Russian / English interface (i18n)
- Secret URL entry — panel is invisible without the token
- JWT auth with invite-link system
- Live metrics: traffic, online peers, system load, memory
- SSE-based real-time update stream with progress display

### Node Management
- One-command node onboarding: copy and paste from panel → node registers automatically
- Per-node health monitoring (online / offline / pending)
- Per-node protocol badge (WG · AWG · Xray)
- Agent reports: CPU load, memory, uptime, peer count, rx/tx bytes

### VPN Protocols
| Protocol | Obfuscation | DPI resistance | Notes |
|---|---|---|---|
| WireGuard | ✗ | low | Standard WG, maximum compatibility |
| AmneziaWG | ✓ | high | Junk packets + handshake padding + XOR |
| Xray-Reality | ✓ | very high | TLS fingerprint mimicry |

### Peer Management
- Create / enable / disable peers per node
- Download `.conf` (standard WG) or `.vwg` (obfuscated AmneziaWG) config
- Live traffic bars per peer
- QR code export (coming soon)

---

## 🏗 Architecture

```
┌─────────────────────────────────────────┐
│              Your Browser               │
└──────────────────┬──────────────────────┘
                   │ HTTPS /:entry-token
┌──────────────────▼──────────────────────┐
│          Panel VPS (nginx + Docker)     │
│  ┌─────────────┐  ┌────────────────┐   │
│  │  Frontend   │  │  Backend API   │   │
│  │  React/Vite │  │  Go + chi      │   │
│  │  port 443   │  │  port 8080     │   │
│  └─────────────┘  └───────┬────────┘   │
│                           │ mTLS       │
└───────────────────────────┼────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
┌─────────▼────┐  ┌─────────▼────┐  ┌────────▼─────┐
│  Node VPS 1  │  │  Node VPS 2  │  │  Node VPS 3  │
│  Agent :7443 │  │  Agent :7443 │  │  Agent :7443 │
│  WireGuard   │  │  AmneziaWG   │  │  Xray-Reality│
└──────────────┘  └──────────────┘  └──────────────┘
```

### Components

| Directory | Language | Role |
|---|---|---|
| `frontend/` | React + TypeScript + Vite | Web UI |
| `backend/` | Go 1.22, chi, pgx | REST API, JWT, mTLS CA |
| `agent/` | Go 1.22 | Node agent (runs on each VPS) |
| `clients/void-wg/` | Go 1.22 | Obfuscated WG client (AmneziaWG-style) |
| `clients/void-wg-d/` | Go 1.22 | Standard WG client with kill switch |

---

## 🖥 VPN Clients

Two CLI clients are included for connecting end-users.

### void-wg-d — Standard WireGuard client

Full wg-quick replacement with kill switch support. Reads standard `.conf` files — compatible with any WireGuard server.

```bash
# Import a config
sudo void-wg-d import ~/vpn.conf

# Connect with kill switch
sudo void-wg-d up vpn --kill-switch

# Status
void-wg-d status

# Disconnect
sudo void-wg-d down vpn
```

### void-wg — Obfuscated client (AmneziaWG-style)

Reads `.vwg` configs — a superset of `.conf` that adds obfuscation parameters. Implements the AmneziaWG protocol: junk packets before handshake, padding on init messages, XOR on WireGuard message types.

```bash
# Import standard WG config and upgrade to .vwg
sudo void-wg upgrade ~/vpn.conf

# Connect (obfuscation on by default if params present)
sudo void-wg up vpn

# Status with obfs info
void-wg status

# Disconnect
sudo void-wg down vpn
```

### Client comparison

| Feature | void-wg-d | void-wg |
|---|---|---|
| Config format | `.conf` (standard) | `.vwg` (superset of .conf) |
| WG protocol | standard | AmneziaWG obfuscation |
| DPI resistance | ✗ | ✓ (junk packets + XOR) |
| Kill switch | ✓ | ✓ |
| Import .conf | ✓ | ✓ (auto-upgrade) |
| Junk packets (Jc/Jmin/Jmax) | ✗ | ✓ |
| Handshake padding (S1/S2) | ✗ | ✓ |
| Message XOR (H1-H4) | ✗ | ✓ |
| Use case | Standard WG servers | void-wg / AmneziaWG servers |

### .vwg config format

A `.vwg` file is a standard `.conf` file with an optional `[Obfuscation]` section:

```ini
[Interface]
PrivateKey = <key>
Address = 10.8.0.2/32
DNS = 1.1.1.1

[Obfuscation]
Enabled = true
Jc    = 4
Jmin  = 40
Jmax  = 70
S1    = 0
S2    = 0
H1    = 1
H2    = 2
H3    = 3
H4    = 4

[Peer]
PublicKey  = <server-pub>
Endpoint   = 1.2.3.4:51821
AllowedIPs = 0.0.0.0/0
```

---

## 🚀 Adding a Node

1. Open the panel → **Servers** → **Add Server**
2. Fill in name, IP, protocol (WG / AWG / Xray)
3. Copy the generated install command — it looks like:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/wester11/v-ui/node/scripts/install-node.sh) \
  --control-url https://1.2.3.4 \
  --node-id abc123 \
  --secret s3cr3t \
  --token agenttoken \
  --protocol amneziawg
```

4. Paste and run on the node VPS — the agent installs, connects back, and the node appears **ONLINE** in ~30 seconds.

---

## 🔒 HTTPS / TLS

### Self-signed (IP mode)

Default for fresh installs on a bare IP. Browser shows a one-time certificate warning.

To regenerate the certificate (e.g. after IP change):

```bash
sudo bash /opt/void-wg/scripts/regen-tls.sh
```

### Let's Encrypt (domain mode)

If you have a domain (e.g. `vpn.example.com`) pointing to the server, choose domain mode during install. Certbot runs automatically, certificate auto-renews via cron.

### Re-configure TLS after install

Run the installer again — it detects the existing config and offers the choice:

```
TLS mode: 1-Keep current  2-Domain cert  3-IP self-signed
```

---

## 📁 Project Structure

```
void-wg/
├── frontend/               # React SPA
│   ├── src/
│   │   ├── pages/          # Dashboard, Servers, Peers, Users, Settings
│   │   ├── components/     # Shared UI components
│   │   ├── api/            # API client + types
│   │   ├── i18n/           # EN/RU translations
│   │   └── styles/         # Global CSS (dark SaaS theme)
│   └── nginx.https.conf.tpl
├── backend/                # Go REST API
│   ├── cmd/api/            # Entrypoint
│   ├── internal/
│   │   ├── domain/         # Business entities
│   │   ├── interfaces/http/ # HTTP handlers (chi)
│   │   ├── repository/     # PostgreSQL queries
│   │   └── service/        # Business logic
│   └── migrations/         # SQL migrations
├── agent/                  # Node agent (Go binary)
│   ├── cmd/agent/          # Entrypoint
│   └── internal/
│       ├── transport/      # WG / AWG / Xray transports
│       ├── wg/             # wg-quick wrapper
│       ├── awg/            # AmneziaWG params
│       └── sysstat/        # /proc + wg show metrics
├── clients/
│   ├── void-wg/            # Obfuscated WG client (.vwg)
│   └── void-wg-d/          # Standard WG client (.conf)
├── scripts/
│   ├── install.sh          # Panel one-command installer
│   ├── install-node.sh     # Node agent installer
│   └── regen-tls.sh        # TLS cert regeneration
├── releases/
│   ├── panel/              # Panel branch release notes
│   ├── node/               # Node branch release notes
│   └── xray/               # Xray branch release notes
├── runtime/                # Runtime mounts (TLS certs, acme)
│   ├── tls/
│   └── agent-ca/
└── docker-compose.yml
```

---

## ⚙️ Environment Variables

Key variables in `.env` (auto-generated by installer, located at `/opt/void-wg/.env`):

| Variable | Description | Example |
|---|---|---|
| `BOOTSTRAP_ADMIN_EMAIL` | Initial admin login | `admin@example.com` |
| `BOOTSTRAP_ADMIN_PASSWORD` | Initial admin password | `changeme` |
| `JWT_SECRET` | JWT signing secret (auto-generated) | `64-char random` |
| `PUBLIC_BASE_URL` | Panel public URL | `https://1.2.3.4` |
| `PANEL_ENTRY_TOKEN` | Secret URL path segment | `abc123def456` |
| `PANEL_HTTPS_PORT` | HTTPS port | `443` |
| `TLS_MODE` | `letsencrypt` or `selfsigned` | `selfsigned` |
| `PANEL_DOMAIN` | Domain for Let's Encrypt | `vpn.example.com` |
| `LOG_LEVEL` | Backend log level | `info` |

---

## 🔧 Node Agent Variables

Set on each node VPS via its own `.env`:

| Variable | Default | Description |
|---|---|---|
| `AGENT_PROTOCOL` | `amneziawg` | `wireguard` / `amneziawg` / `xray` |
| `CONTROL_URL` | — | Panel API URL |
| `AGENT_TOKEN` | — | Auth token for panel ↔ agent |
| `NODE_ID` | — | UUID assigned by panel |
| `SECRET` | — | Shared secret for registration |
| `WG_IFACE` | `wg0` | WireGuard interface name |
| `WG_ADDR` | `127.0.0.1:51820` | Local WG bind address |
| `OBFS_LISTEN` | `:51821` | AWG obfuscation listener |
| `HTTP_LISTEN` | `:7443` | Agent HTTP API port |
| `AWG_JC` | `4` | Junk packet count |
| `AWG_JMIN` | `40` | Junk packet min size |
| `AWG_JMAX` | `70` | Junk packet max size |

---

## ❓ FAQ

**Q: The browser shows a certificate warning after install.**
A: This is expected with the self-signed (IP) cert. Click "Advanced → Proceed". To fix permanently, get a domain and run the installer with Let's Encrypt mode.

**Q: How do I update the panel?**
A: Open Settings → System in the panel UI and click **Check for updates**. Or via SSH:
```bash
cd /opt/void-wg && git pull && docker compose up -d --build
```

**Q: Can I use standard WireGuard clients with this panel?**
A: Yes. Download the `.conf` from the panel → Peers page. Import it into any WireGuard client (official app, `wg-quick`, `void-wg-d`).

**Q: What's the difference between AmneziaWG and standard WireGuard here?**
A: AmneziaWG adds obfuscation on top of the WireGuard UDP packets: random junk packets before the handshake, padding on init messages, and XOR masks on the WireGuard message type field. This makes the traffic look like random UDP, defeating DPI-based blocking.

**Q: How do I add more admins?**
A: Go to Settings → Invites, generate an invite link, and share it. The invited user gets admin access.

**Q: How do I backup the database?**
```bash
docker exec voidwg-postgres-1 pg_dump -U voidwg voidwg > backup.sql
```

---

## 📄 License

MIT © 2024 [wester11](https://github.com/wester11)

---

<div align="center">
Made with ❤️ and a lot of Go
</div>

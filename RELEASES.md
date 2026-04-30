# Releases Layout

This repository uses dedicated release branches with files stored under `releases/<name>/`.

## `panel` branch

Folder: `releases/panel/`

Contains:

- `frontend/`
- `backend/`
- `docker-compose.yml`
- `scripts/install.sh`
- `scripts/update.sh`
- `scripts/regen-tls.sh`
- `scripts/renew-cert.sh`
- `scripts/uninstall.sh`
- `scripts/v-wg.sh`

## `xray` branch

Folder: `releases/xray/`

Contains:

- `backend/domain/config.go`
- `backend/pkg/xray/`
- `backend/infrastructure/xray/`
- `backend/usecase/server.go` (cascade)
- `agent/internal/` xray part

## `node` branch

Folder: `releases/node/`

Contains:

- `agent/`
- `scripts/install-node.sh`

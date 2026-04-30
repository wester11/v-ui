# Node — Changelog

Ветка `node` содержит изменения агента VPN-ноды.

## Что сюда входит
- `agent/` — Go агент (WireGuard / AmneziaWG / Xray sidecar)
- `agent/Dockerfile`
- `scripts/install-node.sh` — установщик ноды на VPS
- `scripts/uninstall.sh`

## Что НЕ входит
- UI панели — это ветка `panel`
- Xray-конфиги — это ветка `xray`

---

## v0.3.0 — 2026-04-30
### Исправлено
- `agent/Dockerfile`: `go mod tidy` вместо `go mod download`
  — решена проблема отсутствия `go.sum` при сборке
- Образ: `golang:1.22-alpine` → multi-stage, итоговый `alpine:3.19`
- Добавлен `docker-cli` в runtime образ (для xray-режима restart sidecar)

### Агент принимает
- `CONTROL_URL` — URL панели управления
- `NODE_ID` — UUID ноды
- `SECRET` — секрет для heartbeat аутентификации
- `AGENT_INSECURE_TLS=true` — для самоподписанных сертификатов панели

### install-node.sh
- Принимает base64-токен: `bash <(curl -Ls URL) <TOKEN>`
- Декодирует: `CONTROL_URL NODE_ID SECRET`
- Backward compat: поддерживает старые флаги `--control-url= --node-id= --secret=`
- Устанавливает Docker через официальный apt repo
- Проверяет связь с панелью через `curl -fsk`

## Roadmap
- [ ] Авто-обновление агента через pull-mode job queue
- [ ] Метрики трафика (rx/tx per peer)
- [ ] Поддержка AmneziaWG
- [ ] Health check endpoint на агенте

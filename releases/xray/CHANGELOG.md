# Xray — Changelog

Ветка `xray` содержит изменения протокола Xray/VLESS/XTLS.

## Что сюда входит
- `backend/internal/domain/xray*.go` — доменные модели Xray
- `backend/internal/application/usecase/*xray*.go` — бизнес-логика
- `backend/internal/infrastructure/xray/` — генерация конфигов
- `agent/internal/xray/` — запуск xray на ноде
- Шаблоны конфигурации VLESS/XTLS/Reality

## Что НЕ входит
- UI панели — это ветка `panel`
- WireGuard/AmneziaWG агент — это ветка `node`

---

## v0.3.0 — 2026-04-30
### Добавлено
- Cascade interconnect: upstream → downstream VLESS-handshake
- `ensureSystemClient` — идемпотентная добавка системного клиента в XrayConfig
- `normalizeCascadeRules` — nормализация правил маршрутизации geoip
- `XrayCascadeRule`: match + outbound (proxy/direct)
- `XraySystemClient`: служебные VLESS-клиенты для межнодового трафика
- `splitEndpointHostPort` helper
- Reality & XTLS-rprx-vision flow поддержка

## Roadmap
- [ ] VLESS WebSocket fallback
- [ ] Auto-rotate Reality keys
- [ ] Per-node bandwidth limit через xray policy
- [ ] Subscription link генератор (клиентские конфиги)

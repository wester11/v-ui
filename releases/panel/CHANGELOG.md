# Panel — Changelog

Ветка `panel` содержит изменения фронтенда и бэкенда панели управления.

## Что сюда входит
- `frontend/` — React + Vite UI
- `backend/` — Go API (chi router, PostgreSQL)
- `scripts/install.sh`, `scripts/regen-tls.sh`, `scripts/update.sh`
- `docker-compose.yml`, `frontend/nginx.*.conf*`

## Что НЕ входит
- `agent/` — это ветка `node`
- Xray/VLESS конфигурации — это ветка `xray`

---

## v0.3.0 — 2026-04-30
### Добавлено
- Полный визуальный редизайн: glass morphism, dot grid, glow-эффекты
- Animated count-up счётчики на Dashboard
- SSE streaming прогресс обновления панели (`/api/v1/admin/system/update/stream`)
- Endpoint для версии и апдейта системы
- PATCH endpoints: включение/отключение peer и user
- Секретный URL-токен для защиты панели (`PANEL_ENTRY_TOKEN`)
- Установщик нод через base64-токен (install-node.sh на GitHub)
- `regen-tls.sh` — регенерация TLS без переустановки
- Исправлен `ERR_SSL_KEY_USAGE_INCOMPATIBLE` (добавлен `digitalSignature`)
- SSE nginx location с `proxy_read_timeout 3600s`, `proxy_buffering off`
- i18n EN/RU полностью
- Авто-поллинг onboarding modal каждые 5 секунд

## v0.2.0 — предыдущая сессия
### Добавлено
- SVG иконки sidebar
- Sidebar expandable Серверы + sub-nav
- Users/Peers enable/disable toggle
- Traffic bar на странице Peers
- Fleet health + redeploy all

# void-wg

> [!IMPORTANT]
> Canonical repository: https://github.com/wester11/v-ui
>
> Use only v-ui links for install/clone.
> void_wg may be outdated and can break installs.

Production-ready WireGuard VPN management system: REST API + JWT, React UI,
агенты на узлах, базовая обфускация UDP-трафика, метрики Prometheus,
Docker-ready, one-click установка.

> Архитектура — clean / hexagonal. Подробнее в [`ARCHITECTURE.md`](./ARCHITECTURE.md)
> и [`STRUCTURE.md`](./STRUCTURE.md).

## One-click установка

```bash
bash <(curl -Ls https://raw.githubusercontent.com/wester11/v-ui/main/scripts/install.sh)
```

Скрипт:

* проверит ОС (Ubuntu 20.04+ / Debian 11+) и root,
* поставит зависимости: `docker`, `docker compose`, `git`, `openssl`, `ufw`, `wireguard-tools`, `certbot`,
* склонирует репозиторий в `/opt/void-wg`,
* сгенерирует `.env` со случайными секретами и паролем админа,
* **спросит, какой TLS использовать** (1 = self-signed по IP, 2 = Let's Encrypt по домену, 3 = HTTP-only),
* для Let's Encrypt — попросит домен и email, выдаст сертификат через `certbot --standalone`,
* откроет нужные порты в `ufw` (включая 80/tcp и 443/tcp),
* поставит systemd-юнит `void-wg.service`,
* для Let's Encrypt — поставит таймер `void-wg-renew.timer`, обновляющий cert дважды в сутки,
* соберёт и поднимет docker compose,
* выведет учётные данные.

Пример вывода (TLS_MODE=letsencrypt, домен `vpn.example.com`):

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Installation complete!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Panel:    https://vpn.example.com
  TLS mode: letsencrypt
  Login:    admin@local
  Password: x8F2kL9pQrT4mN7Z

  Files:    /opt/void-wg
  .env:     /opt/void-wg/.env  (mode 600 — back it up!)
  TLS dir:  /opt/void-wg/runtime/tls
  Logs:     docker compose -f /opt/void-wg/docker-compose.yml logs -f
  Update:   sudo bash /opt/void-wg/scripts/update.sh
  Renew:    sudo bash /opt/void-wg/scripts/renew-cert.sh   (Let's Encrypt only)
  Remove:   sudo bash /opt/void-wg/scripts/uninstall.sh

  systemd:  systemctl status void-wg
  TLS timer: systemctl status void-wg-renew.timer
```

Скрипт идемпотентен — повторный запуск не пересоздаёт пароль и не ломает установку.

### Параметры через env

```bash
PANEL_PORT=9000 ADMIN_EMAIL=root@example.com \
bash <(curl -Ls https://raw.githubusercontent.com/wester11/v-ui/main/scripts/install.sh)
```

| Переменная          | По умолчанию                                  | Назначение                                       |
|---------------------|-----------------------------------------------|--------------------------------------------------|
| `INSTALL_DIR`       | `/opt/void-wg`                                | куда ставить                                     |
| `PANEL_HTTP_PORT`   | `80`                                          | порт HTTP (редирект на HTTPS, ACME challenge)    |
| `PANEL_HTTPS_PORT`  | `443`                                         | порт HTTPS-панели                                |
| `WG_PORT`           | `51820`                                       | UDP-порт WireGuard                               |
| `OBFS_PORT`         | `51821`                                       | UDP-порт обфускации                              |
| `ADMIN_EMAIL`       | `admin@local`                                 | email админа                                     |
| `TLS_MODE`          | спрашивает интерактивно                       | `selfsigned`, `letsencrypt` или `none`           |
| `PANEL_DOMAIN`      | спрашивает интерактивно (если `letsencrypt`)  | домен для Let's Encrypt                          |
| `LE_EMAIL`          | спрашивает интерактивно                       | email для уведомлений Let's Encrypt              |
| `REPO_URL`          | `https://github.com/wester11/v-ui.git`     | откуда клонировать                               |
| `REPO_BRANCH`       | `main`                                        | какая ветка                                      |

Не-интерактивный пример:

```bash
TLS_MODE=letsencrypt PANEL_DOMAIN=vpn.example.com LE_EMAIL=ops@example.com \
bash <(curl -Ls https://raw.githubusercontent.com/wester11/v-ui/main/scripts/install.sh)
```

## CLI: `v-wg`

После установки в системе появляется команда `v-wg` (символическая ссылка на `scripts/v-wg.sh`). Без аргументов — интерактивное TUI-меню в стиле `x-ui`:

```
╔──────────────────────────────────────────────────╗
│   void-wg Panel Management Script                │
│   0. Exit                                        │
│──────────────────────────────────────────────────│
│   1. Install / Reinstall                         │
│   2. Update                                      │
│   3. Uninstall                                   │
│──────────────────────────────────────────────────│
│   4. Reset admin password                        │
│   5. View current credentials                    │
│   6. View settings                               │
│   7. Change panel ports                          │
│──────────────────────────────────────────────────│
│   8. Start                                       │
│   9. Stop                                        │
│  10. Restart                                     │
│  11. Status                                      │
│  12. Live logs                                   │
│──────────────────────────────────────────────────│
│  13. Enable autostart                            │
│  14. Disable autostart                           │
│──────────────────────────────────────────────────│
│  15. TLS: renew certificate now                  │
│  16. TLS: switch mode (IP / domain / none)       │
│  17. Firewall management (ufw)                   │
│──────────────────────────────────────────────────│
│  18. Enable BBR                                  │
╚──────────────────────────────────────────────────╝
Panel state: Running (4/4)
Autostart:   Enabled
TLS mode:    Let's Encrypt (vpn.example.com)
Please enter your selection [0-18]:
```

Поддерживаются аргументы-алиасы:

```bash
sudo v-wg status            # = 11
sudo v-wg restart           # = 10
sudo v-wg logs              # = 12
sudo v-wg renew             # = 15
sudo v-wg reset-pass        # = 4
sudo v-wg ports             # = 7
sudo v-wg tls               # = 16
```

## TLS / сертификаты

`install.sh` поддерживает три режима терминации TLS на nginx внутри `frontend`-контейнера:

* **`selfsigned`** (по умолчанию для IP) — `openssl` генерирует cert с `CN=<server-ip>` и SAN `IP:<server-ip>`, валидность 10 лет. Браузер покажет предупреждение, но соединение шифруется. Никакой авто-rotation не нужен; `renew-cert.sh` всё равно перевыпустит cert, если до конца жизни остаётся <30 дней.
* **`letsencrypt`** — `certbot certonly --standalone` выдаёт настоящий cert по http-01 challenge. Cert копируется в `runtime/tls/` и подключается nginx'ом. Автообновление — systemd-таймером `void-wg-renew.timer` (дважды в сутки, со случайной задержкой 0–2ч). При фактическом обновлении frontend останавливается на ~10 секунд (pre/post-hook у certbot), на no-op-проверках — ничего не трогает.
* **`none`** — HTTP-only, только для разработки.

Файлы и пути:

```
/opt/void-wg/runtime/
├── nginx.conf       # сгенерированный install.sh — HTTP-only или HTTPS+redirect
├── tls/
│   ├── fullchain.pem
│   └── privkey.pem  # mode 600
└── acme-www/        # webroot для http-01 challenge
```

Ручное обновление сертификата:

```bash
sudo bash /opt/void-wg/scripts/renew-cert.sh
journalctl -u void-wg-renew.service --since '1 day ago'   # лог последнего ранна
systemctl status void-wg-renew.timer                       # когда следующий запуск
```

Сменить домен / переключить режим:

```bash
sudo TLS_MODE=letsencrypt PANEL_DOMAIN=newvpn.example.com \
     bash /opt/void-wg/scripts/install.sh   # idempotent: пересоберёт TLS-конфиг
```

## Update / Uninstall

```bash
sudo bash /opt/void-wg/scripts/update.sh      # git pull + rebuild + restart
sudo bash /opt/void-wg/scripts/uninstall.sh   # снос с подтверждением
```

## Запуск через docker compose (без install.sh)

```bash
git clone https://github.com/wester11/v-ui.git
cd v-ui
cp .env.example .env
# отредактируйте секреты в .env (или просто запустите ./scripts/install.sh)
docker compose up -d --build
```

## API

Полная спецификация — [`backend/api/openapi.yaml`](./backend/api/openapi.yaml).
Базовые операции:

```bash
# 1. Login
TOKEN=$(curl -s http://SERVER_IP:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@local","password":"x8F2kL9pQrT4mN7Z"}' | jq -r .access_token)

# 2. Создать пользователя
curl -s http://SERVER_IP:8080/api/v1/users \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"email":"alice@example.com","password":"strong-pass","role":"user"}'

# 3. Зарегистрировать VPN-ноду
curl -s http://SERVER_IP:8080/api/v1/servers \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"de-1","endpoint":"vpn.example.com:51820","listen_port":51820,
       "subnet":"10.10.0.0/24","dns":["1.1.1.1"],"obfs_enabled":true}'

# 4. Создать peer и сразу получить wg-quick конфиг
curl -s http://SERVER_IP:8080/api/v1/peers \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"server_id":"<UUID>","name":"my-iphone"}' | jq -r .config > my-iphone.conf

# 5. Удалить пользователя / peer'а
curl -X DELETE http://SERVER_IP:8080/api/v1/users/<UUID>  -H "Authorization: Bearer $TOKEN"
curl -X DELETE http://SERVER_IP:8080/api/v1/peers/<UUID>  -H "Authorization: Bearer $TOKEN"
```

## Подключение дополнительных нод

В UI: **Servers → Register**, скопировать `agent_token`. Затем на нужном хосте:

```bash
docker run -d --name void-wg-agent --restart=unless-stopped \
  --cap-add NET_ADMIN --network host \
  -e CONTROL_URL=http://control.example.com:8080 \
  -e AGENT_TOKEN=<token> \
  -e WG_IFACE=wg0 \
  -e OBFS_LISTEN=:51821 \
  -e WG_ADDR=127.0.0.1:51820 \
  -e OBFS_PSK=$(openssl rand -hex 32) \
  ghcr.io/wester11/void-wg-agent:latest    # либо собрать локально из ./agent
```

## Веб-панель

* **Dashboard** — счётчики users / peers / servers / суммарный трафик, статусы агентов в реальном времени (poll 15s).
* **Peers** — поиск, модалка создания, QR-код для wg-quick конфига, копирование/скачивание `.conf`, badges статусов.
* **Servers** — регистрация ноды, agent_token с маскировкой и copy-to-clipboard, страница детали `/servers/:id` со сводкой по трафику и пирам.
* **Users** — управление учётками с цветными role-badges (admin / operator / user), нельзя удалить себя.
* **Profile** — смена своего пароля (`PATCH /api/v1/me/password`), просмотр своих данных.
* Sidebar-навигация с фильтрацией по роли, dropdown user-menu, тосты, скелетоны загрузки, тёмная тема.

## Метрики и логи

* `GET /metrics` — Prometheus (доступ только из private nets, см. `frontend/nginx.conf`).
* Логи всех сервисов — JSON в stdout: `docker compose logs -f api`.
* Опциональный Prometheus-сборщик — поднимается профилем:
  `docker compose --profile metrics up -d prometheus`.

## Production checklist

- [ ] Поставить TLS-терминатор перед панелью (Caddy/Traefik/Nginx + Let's Encrypt).
- [ ] Включить mTLS между control-plane и агентами (`AGENT_INSECURE_TLS=false`).
- [ ] Настроить регулярный backup `pgdata` тома и `.env` (там зашифрованные приватники).
- [ ] Сменить дефолтный `admin@local` на реальный email и удалить bootstrap-юзера.
- [ ] Включить rate-limit `/api/v1/auth/login` на ingress.
- [ ] Подключить алерты на падение агентов (`voidwg_agents_online == 0`).

## Разработка

```bash
# Backend
cd backend && go mod tidy && go run ./cmd/api

# Frontend
cd frontend && npm install && npm run dev   # http://localhost:5173

# Запушить в свой fork
GH_TOKEN=ghp_xxx GH_USER=<me> GH_REPO=v-ui bash scripts/publish-to-github.sh
```

## Лицензия

MIT

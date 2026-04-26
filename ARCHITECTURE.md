# Void-WG — Архитектура

## 1. Общая схема

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Web UI (React + TS)                          │
│        (login, users, peers, servers, конфиги, метрики)              │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ HTTPS / JWT
┌──────────────────────────────▼──────────────────────────────────────┐
│                     Control Plane (Go API)                           │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │  Interfaces (HTTP)                                             │ │
│  │   ├── REST handlers (chi)                                      │ │
│  │   ├── DTO ⇄ domain mapping                                     │ │
│  │   ├── OpenAPI 3.1 (swagger UI)                                 │ │
│  │   └── middleware: auth, request-id, recover, metrics           │ │
│  ├────────────────────────────────────────────────────────────────┤ │
│  │  Application (use-cases)                                       │ │
│  │   ├── AuthService  (login / refresh / me)                      │ │
│  │   ├── UserService  (CRUD + roles)                              │ │
│  │   ├── PeerService  (provision / revoke / config)               │ │
│  │   ├── ServerService (register / heartbeat / sync)              │ │
│  │   └── AgentBus     (push commands → agents)                    │ │
│  ├────────────────────────────────────────────────────────────────┤ │
│  │  Domain (pure)                                                 │ │
│  │   ├── User, Peer, Server, Session                              │ │
│  │   ├── value-objects: WGKey, IPv4, AllowedIPs                   │ │
│  │   └── policy / errors                                          │ │
│  ├────────────────────────────────────────────────────────────────┤ │
│  │  Ports (interfaces)                                            │ │
│  │   ├── UserRepo, PeerRepo, ServerRepo                           │ │
│  │   ├── KeyGenerator, Hasher, TokenIssuer                        │ │
│  │   └── AgentTransport                                           │ │
│  ├────────────────────────────────────────────────────────────────┤ │
│  │  Infrastructure (adapters)                                     │ │
│  │   ├── postgres (pgx)                                           │ │
│  │   ├── wgkeygen (curve25519)                                    │ │
│  │   ├── jwtauth  (HS256/EdDSA)                                   │ │
│  │   ├── grpc/http2 agent transport                               │ │
│  │   ├── prometheus / zerolog                                     │ │
│  │   └── obfuscation (padding + noise)                            │ │
│  └────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ mTLS + token (gRPC)
        ┌──────────────────────┼─────────────────────┐
        ▼                      ▼                     ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│  Agent (node-1)  │  │  Agent (node-2)  │  │  Agent (node-N)  │
│ ─ wg netlink ctl │  │ ─ wg netlink ctl │  │ ─ wg netlink ctl │
│ ─ obfs proxy     │  │ ─ obfs proxy     │  │ ─ obfs proxy     │
│ ─ heartbeat      │  │ ─ heartbeat      │  │ ─ heartbeat      │
└──────────────────┘  └──────────────────┘  └──────────────────┘
        │                      │                     │
        ▼                      ▼                     ▼
   wg0 (UDP/51820)      wg0 (UDP/51820)        wg0 (UDP/51820)
```

## 2. Принципы

* **Hexagonal / Clean** — domain не зависит ни от чего; зависимости направлены внутрь.
* **Ports & Adapters** — все внешние интеграции спрятаны за интерфейсами в `application/port`.
* **CQRS-light** — read/write модели одинаковы, но команды и запросы — разные методы use-case.
* **Stateless API** — авторизация через JWT (access 15m + refresh 30d).
* **Multi-tenant ready** — server_id в каждой сущности peer/route.

## 3. Поток создания peer

1. `POST /api/v1/peers` (UI) → handler валидирует DTO.
2. `PeerService.Provision` создаёт доменный `Peer`, генерирует пару ключей через `KeyGenerator`.
3. Resolver подбирает IP из CIDR пула сервера (`ServerRepo`).
4. `PeerRepo.Save` (Postgres, транзакция).
5. `AgentTransport.Apply(server_id, peer)` → агент применяет `wg set` через netlink.
6. Возвращается готовый конфиг `.conf` + QR-код (опционально).

## 4. Обфускация

Реализована как **userspace UDP-proxy** перед `wg0`:

* **Random padding** — каждый исходящий пакет дополняется случайным числом байт (0–N) и помечается флагом длины.
* **Packet shaping** — рандомизация межпакетных интервалов в режиме «idle» (анти-таймнинг-атаки).
* **XOR-obfuscation** — пакет XOR-ится с derived key (HKDF от PSK).
* **Decoy packets** — периодически шлются «шумовые» пакеты случайного размера.

Совместимый клиент-обфускатор поставляется как Go-библиотека и может быть встроен в кастомный мобильный клиент. Для стандартных клиентов агент держит «прозрачный режим» (без обфускации).

## 5. Безопасность

* пароли — argon2id;
* приватные ключи WG **никогда** не покидают сервер/агент в открытом виде; в БД хранится только публичный ключ + зашифрованный (AES-GCM) приватный для генерации `.conf` пользователю;
* mTLS между control-plane и агентами;
* RBAC: `admin`, `operator`, `user`;
* rate-limit на login.

## 6. Observability

* `zerolog` — структурное JSON логирование c request-id;
* `/metrics` Prometheus: HTTP latency, peers active, bytes-in/out, agent heartbeats;
* OpenTelemetry traces (опционально, через OTLP exporter);
* health: `/healthz`, `/readyz`.

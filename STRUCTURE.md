# Структура проекта

```
void_wg/
├── ARCHITECTURE.md
├── STRUCTURE.md
├── README.md
├── docker-compose.yml
├── .env.example
│
├── backend/                         # Control Plane (Go)
│   ├── go.mod
│   ├── go.sum
│   ├── Dockerfile
│   ├── cmd/
│   │   └── api/
│   │       └── main.go              # точка входа API
│   ├── api/
│   │   └── openapi.yaml             # OpenAPI 3.1
│   ├── migrations/
│   │   └── 001_init.sql
│   ├── internal/
│   │   ├── domain/                  # ядро (без зависимостей)
│   │   │   ├── user.go
│   │   │   ├── peer.go
│   │   │   ├── server.go
│   │   │   └── errors.go
│   │   ├── application/
│   │   │   ├── port/                # интерфейсы (ports)
│   │   │   │   ├── repository.go
│   │   │   │   ├── crypto.go
│   │   │   │   └── transport.go
│   │   │   └── usecase/             # use-cases
│   │   │       ├── auth.go
│   │   │       ├── user.go
│   │   │       ├── peer.go
│   │   │       └── server.go
│   │   ├── infrastructure/          # адаптеры
│   │   │   ├── persistence/
│   │   │   │   ├── postgres.go
│   │   │   │   ├── user_repo.go
│   │   │   │   ├── peer_repo.go
│   │   │   │   └── server_repo.go
│   │   │   ├── wg/
│   │   │   │   └── keygen.go
│   │   │   ├── jwtauth/
│   │   │   │   └── jwt.go
│   │   │   ├── obfuscation/
│   │   │   │   └── obfuscator.go
│   │   │   ├── transport/
│   │   │   │   └── agent_grpc.go
│   │   │   ├── logger/
│   │   │   │   └── logger.go
│   │   │   └── metrics/
│   │   │       └── metrics.go
│   │   └── interfaces/
│   │       └── http/
│   │           ├── router.go
│   │           ├── server.go
│   │           ├── handler/
│   │           │   ├── auth.go
│   │           │   ├── user.go
│   │           │   ├── peer.go
│   │           │   └── server.go
│   │           ├── middleware/
│   │           │   ├── auth.go
│   │           │   ├── recover.go
│   │           │   ├── requestid.go
│   │           │   └── metrics.go
│   │           └── dto/
│   │               └── dto.go
│   └── pkg/
│       └── wireguard/
│           └── config.go            # генерация .conf
│
├── agent/                           # Data plane agent (Go)
│   ├── go.mod
│   ├── Dockerfile
│   ├── cmd/agent/main.go
│   └── internal/
│       ├── wg/manager.go            # netlink wrapper
│       ├── obfs/proxy.go            # UDP proxy с обфускацией
│       └── client/control.go        # клиент к control-plane
│
└── frontend/                        # React + TS
    ├── package.json
    ├── vite.config.ts
    ├── tsconfig.json
    ├── Dockerfile
    ├── index.html
    └── src/
        ├── main.tsx
        ├── App.tsx
        ├── api/client.ts
        ├── store/auth.ts
        ├── types/index.ts
        ├── components/
        │   ├── Layout.tsx
        │   └── ProtectedRoute.tsx
        └── pages/
            ├── Login.tsx
            ├── Peers.tsx
            ├── Servers.tsx
            └── Users.tsx
```

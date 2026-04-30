# Contributing

Thanks for improving `v-ui`.

## Local setup

1. Install:
   - Go 1.22+
   - Node.js 20+
   - Docker + Docker Compose
2. Clone and enter repo.
3. Start stack when needed:

```bash
docker compose up -d --build
```

## Development areas

- `backend/` control plane (Go)
- `agent/` node agent (Go)
- `frontend/` React + TypeScript UI
- `scripts/` install/update scripts
- `releases/` branch release bundles

## Quality gates

- Backend: `cd backend && go test ./...`
- Agent: `cd agent && go test ./...`
- Frontend: `cd frontend && npm ci && npm run build`

## Pull requests

1. Create a branch from `main`
2. Keep PR scope focused
3. Fill PR template
4. Wait for CI to pass before merge

## Commit style

Recommended style:

- `feat: ...`
- `fix: ...`
- `docs: ...`
- `chore: ...`

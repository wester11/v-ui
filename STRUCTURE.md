# Project Structure

```text
v-ui/
├── README.md
├── STRUCTURE.md
├── RELEASES.md
├── ARCHITECTURE.md
├── .editorconfig
├── .gitattributes
├── .env.example
├── docker-compose.yml
│
├── backend/                    # Control plane (Go)
├── agent/                      # Node agent (Go)
├── frontend/                   # Web panel (React + TS)
├── scripts/                    # Install/update helpers
├── deploy/                     # Deployment templates/files
├── runtime/                    # Runtime assets
│
└── releases/                   # Branch-specific release bundles
    ├── panel/
    ├── xray/
    └── node/
```

## Branch Model

- `main`: main development branch.
- `panel`: panel release bundle under `releases/panel/`.
- `xray`: xray-related release bundle under `releases/xray/`.
- `node`: node-agent release bundle under `releases/node/`.

Details for each release branch are in `RELEASES.md`.

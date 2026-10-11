# VietPanel License Server

Self-hosted license validation + management for VietPanel. Speaks the same wire
protocol the panel expects (`POST licensekey+ip` → `<status>Active</status>`), so
panels validate keys against **your** infrastructure — no vendor phone-home.

## Features

- `POST /enterprise/index.php` — validation endpoint every VietPanel component calls, answering
  the upstream WHMCS tags (`status`, `registeredname`, `companyname`, `email`, `productname`,
  `regdate`, `nextduedate`, `billingcycle`, `validip`) so the admin `/license` page has real data
- Web manager UI — login with `LICENSE_ADMIN_TOKEN`, issue / revoke keys, IP binding
- JSON API — `POST|GET /api/keys` with `Authorization: Bearer $LICENSE_ADMIN_TOKEN`
- Keys auto-bind to the first IP that validates; mismatches are denied
- Single static binary, JSON file store (no database required)

## Run with Docker Compose

```bash
git clone https://github.com/trongtri1512/vietpanel.git
cd vietpanel/license-server
cp .env.example .env        # set LICENSE_ADMIN_TOKEN + LICENSE_PUBLIC_URL
docker compose up -d --build
```

UI: `http://<host>:2095/` · validate endpoint: `http://<host>:2095/enterprise/index.php`

## Run with Dokploy

1. Dokploy → **Applications → Create Application**
2. Source: Git provider / raw URL `https://github.com/trongtri1512/vietpanel.git`, branch `main`
3. **Build Type:** Dockerfile · **Dockerfile path:** `license-server/Dockerfile` · **Build context:** `license-server` (or set root dir to `license-server`)
4. Ports: expose `2095` (application) → publish e.g. `443` behind your Dokploy domain/traefik
5. Environment variables:
   - `LICENSE_ADMIN_TOKEN` = a long random secret
   - `LICENSE_PUBLIC_URL` = the public URL Dokploy gives the app (e.g. `https://license.yourdomain.com`)
6. Volumes: container path `/data` (persistent) — stores `licenses.json`
7. Deploy.

## Point a panel at this server

In the panel's `/etc/openpanel/openpanel/conf/openpanel.config`:

```ini
[LICENSE]
key=<issued-key>
license_server_url=https://license.yourdomain.com/enterprise/index.php
```

Issue a key from the web UI (or `curl -X POST $URL/api/keys -H "Authorization: Bearer $TOKEN" -d '{"plan":"enterprise","owner":"Customer A","email":"ops@campa.com"}'`), then restart the panel.

`license_server_url` is the single switch for the whole stack: the panel reads it at boot, VietAdmin overrides its built-in default with it (`admin/cmd/admin/main.go`), and `vietcli license` picks it up from the same block — so a key validates against your server everywhere instead of being checked against the vendor's.

## Env vars

| Var | Default | Purpose |
|---|---|---|
| `LICENSE_DB` | `licenses.json` | key store path (use `/data/licenses.json` in Docker) |
| `LICENSE_ADDR` | `:2095` | listen address |
| `LICENSE_ADMIN_TOKEN` | – | required for UI + management API |
| `LICENSE_PUBLIC_URL` | `http://127.0.0.1:2095` | base URL shown in UI hints |

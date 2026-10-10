# VietPanel

Web hosting control panel — each user gets an isolated container environment (web server, database, network, storage), VPS-grade security on shared hardware.

- **VietPanel** — user control panel (port 2083)
- **VietAdmin** — admin control panel (port 2087)
- **VietCLI** — bash CLI for server management

## Quick Start

```bash
# Start user panel (SQLite mode for dev)
cd panel
DATA_DIR=./data DATA_DIR_SQLITE=true go run ./cmd/panel/

# Start admin panel
cd admin
DATA_DIR=./data go run ./cmd/admin/
```

- User panel: http://localhost:2083 (default: demo / demo123)
- Admin panel: http://localhost:2087 (default: admin / admin123)

## Architecture

- Go 1.26 + html/template + Tailwind CSS + Alpine.js
- Single binary with go:embed for all static assets
- gorilla/csrf for CSRF protection
- SQLite for dev, MySQL for production
- Bash CLI dispatcher (dash-to-slash path mapping)
- Podman per-user containers
- Caddy reverse proxy with auto vhost config

## Project Structure

```
vietpanel/
├── panel/          # User panel (port 2083)
│   ├── cmd/panel/  # Entry point
│   ├── internal/
│   │   ├── app/    # App struct, wiring
│   │   ├── auth/   # Session auth, CSRF
│   │   ├── config/ # Config loading
│   │   ├── core/   # CLI executor, DB, i18n
│   │   ├── modules/# 20 feature modules
│   │   └── web/    # Template rendering
│   └── static/
├── admin/          # Admin panel (port 2087)
│   ├── cmd/admin/
│   ├── internal/
│   │   ├── auth/   # Admin auth
│   │   ├── db/     # SQLite admin DB
│   │   ├── handlers/ # Admin modules
│   │   └── webtemplates/
│   └── static/
├── cli/            # Bash CLI (vietcli)
│   ├── vietcli     # Dispatcher
│   ├── lib/        # Shared functions
│   └── user/       # User commands
├── e2e/            # Playwright E2E tests
└── playwright.config.ts
```

## Features

### User Panel
Dashboard, Websites, Domains, Databases, File Manager, Backups, DNS, Email, FTP, Docker Containers, Service Monitor, Cron Jobs, PHP/Node.js/Python runtime selectors, Cache, Security, Disk Usage, Auto Installer (WordPress, Joomla, Drupal, Nextcloud, Moodle, n8n, Laravel, NodeBB, Mautic, Matomo)

### Admin Panel
Dashboard, User Management, Hosting Plans, Domain Management, Email Server, Service Monitoring, Firewall, Server Settings, Podman Containers, Caddy Reverse Proxy, DNS (BIND9), Mail Server, FTP Server

## Development

```bash
# E2E tests
npx playwright test

# Go tests
cd panel && go test ./...
cd admin && go test ./...

# Build binaries
cd panel && go build -o panel ./cmd/panel/
cd admin && go build -o admin ./cmd/admin/
```

## License

MIT

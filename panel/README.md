# panel/

VietPanel — user control panel (Go, port 2083).

- `cmd/panel/main.go` — entrypoint
- `internal/app/` — App struct: config, session, db pool, i18n
- `internal/core/` — db, podman manager, validators
- `internal/auth/` — session, JWT, CSRF, 2FA
- `internal/modules/` — mỗi feature 1 module: `Register(mux, app)`
- `internal/web/` — templates, layout, sidebar
- `static/` — Tailwind src/dist, JS, vendor libs

Triển khai: container (podman), Caddy terminate TLS phía trước.

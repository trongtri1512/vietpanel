# admin/

VietAdmin — administration panel (Go, port 2087).

- `cmd/admin/main.go` — entrypoint
- `internal/handlers/` — page + JSON handlers
- `internal/auth/` — session, CSRF, 2FA, passkeys
- `internal/db/` — SQLite (admin accounts) + MySQL (panel data)
- `internal/webtemplates/` — ~90 templates
- `internal/license/` — license check + gating (P9)

Triển khai: systemd service trên host, không container.

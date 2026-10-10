# VietPanel — Kế hoạch rebuild hosting control panel

> Ngày lập: 2026-10-10
> Tham chiếu: mô phỏng kiến trúc + giao diện của OpenPanel (openpanel.com) — **viết code mới hoàn toàn, không copy nguyên văn**.

## 1. Mục tiêu & phạm vi

Xây dựng hosting control panel gồm 2 panel web + CLI:

- **VietPanel** (user panel) — port 2083: người dùng cuối quản lý websites, databases, files, backups, DNS, email, FTP, containers...
- **VietAdmin** (admin panel) — port 2087: quản trị users, hosting plans, domains, services, firewall, settings.
- **CLI** (bash) — quản trị server qua terminal, được Go panel gọi qua `exec`.

**Phạm vi:** Full clone (tương đương đầy đủ tính năng bản gốc).
**Tech stack:** Go + `html/template` server-rendered + Tailwind CSS + Alpine.js + `go:embed` 1 binary + Bash CLI (giữ nguyên kiến trúc gốc).
**Giao diện:** mô phỏng phong cách gốc (dark theme, sidebar trái, card layout) nhưng viết lại toàn bộ.

## 2. Kiến trúc mục tiêu

```
                 ┌────────────────────────────────────────────┐
                 │               SERVER (Linux)               │
                 │  Caddy :443 (reverse proxy + SSL)          │
                 │  ├─ VietAdmin :2087 (Go binary, systemd)   │
                 │  └─ VietPanel  :2083 (Go binary, container)│
                 │  MySQL (metadata: users, plans, domains)   │
                 │  SQLite (admin accounts)                   │
                 │  vietcli (bash, /usr/local/vietcli)        │
                 │  BIND9 DNS · mail server · vsftpd          │
                 └──────────────┬─────────────────────────────┘
                     podman unix socket (rootless per-user)
                 ┌──────────────┴─────────────────────────────┐
                 │ USER 1: nginx+php-fpm, mysql, redis...     │
                 │ USER 2: openlitespeed, mariadb...          │
                 └────────────────────────────────────────────┘
```

Luồng dữ liệu: Panel Go → `exec vietcli ...` / trực tiếp `podman` CLI → thao tác container; metadata lưu MySQL; UI gọi `/json/*` (session) và `/api/*` (JWT/API key).

## 3. Cấu trúc repo

```
vietpanel/
├── PLAN.md                  # kế hoạch này
├── panel/                   # user panel (Go, port 2083)
│   ├── cmd/panel/main.go
│   ├── internal/
│   │   ├── app/             # App struct: config, session, db pool, i18n
│   │   ├── core/            # db, podman manager, validators, i18n
│   │   ├── auth/            # session, JWT, CSRF, 2FA, passkeys
│   │   ├── modules/         # ~50 modules, mỗi module: Register(mux, app)
│   │   └── web/             # templates, layout, sidebar
│   ├── static/              # tailwind src/dist, js, vendor (alpine, chart.js...)
│   └── assets.go            # go:embed
├── admin/                   # admin panel (Go, port 2087, systemd service)
│   ├── cmd/admin/main.go
│   ├── internal/{handlers,auth,db,webtemplates,server,license}
│   └── static/
├── cli/                     # bash: dispatcher + user/, domains/, plan/, email/...
├── configuration/           # templates: caddy, bind9, mysql, nginx/apache/litespeed, docker-compose
├── integrations/            # whmcs, fossbilling, cpanel-import...
├── tests/                   # Playwright E2E (panel/ + admin/)
├── website/                 # docs (Docusaurus)
└── scripts/                 # build, install
```

## 4. Các giai đoạn thực hiện

| Phase | Nội dung | Deliverables chính | Ước lượng* |
|---|---|---|---|
| **P1. Nền tảng chung** | Repo + tooling, Tailwind pipeline, `base.html` layout + sidebar (dark theme), config loader, i18n, session/CSRF, login/logout, JWT API auth, 2FA, activity log | Cả 2 panel chạy được login + dashboard trống | 2–3 tuần |
| **P2. CLI core** | Dispatcher `vietcli` (map command→script, log, hooks), thư viện dùng chung (`lib/`), nhóm lệnh `user/`, `plan/`, `domain/` cơ bản | CLI hoạt động độc lập, Go gọi được qua exec | 2 tuần |
| **P3. User panel – core** | Modules theo thứ tự phụ thuộc: **websites/domains** (vhost, SSL, redirect), **mysql** (tạo db/user), **filemanager** (lớn nhất, ~22 templates), **backups**, **dns**, **ftp**, **emails**, **crons**, **services** | User quản lý được 1 website + DB + file từ đầu đến cuối | 6–8 tuần |
| **P4. User panel – nâng cao** | docker (containers + xterm terminal), php (version, phpMyAdmin), nodejs/python/ruby, cache (redis/varnish...), waf, ipblocker, processmanager, diskusage/inodes, trash, goaccess | Đủ ~40/50 modules | 6–8 tuần |
| **P5. Autoinstaller** | wordpress, joomla, drupal, nextcloud, moodle, n8n... (1 module pattern + danh sách app) | Cài 1-click từ panel | 2–3 tuần |
| **P6. Admin panel** | Login + dashboard, **users CRUD** (+ bulk, autologin SSO), **plans/features**, domains (DNS zone, SSL, logs, stats), emails, services (systemd + podman), firewall (CSF), WAF, server ops (reboot, crons, migrate, updates), settings (~90 templates) | Admin quản trị toàn bộ server | 8–10 tuần |
| **P7. Hạ tầng hệ thống** | Per-user Podman provisioning (compose templates, rootless socket), Caddy tự động vhost/SSL, BIND9 zones, mail server, vsftpd, cPanel/CyberPanel import | Sản phẩm chạy thật trên 1 VPS | 4–6 tuần |
| **P8. Chất lượng** | Playwright E2E cho cả 2 panel (~120 spec), Go unit tests, CI (GitHub Actions), docs website, integrations billing (WHMCS/FOSSBilling) | Testable + release-ready | 3–4 tuần |
| **P9. License gating (Enterprise)** | Package `license` (check + backoff + grace period), lệnh `vietcli license`, trang `settings_license` trong admin, cơ chế `enabled_modules` toggle, gating middleware + UI | Một codebase, bật/tắt tính năng bằng license key | 2–3 tuần |

*Ước lượng cho 1 dev + AI hỗ trợ, full-time. Tổng: **~9–12 tháng**; bản MVP (P1–P3 + P6 tối giản) khả thi trong **~2–3 tháng**.

### Trạng thái hiện tại (cập nhật 2026-10-10)

**P1 — HOÀN THÀNH phần nền tảng** (đã chạy + kiểm tra bằng browser):

- **Panel (port 2083)** — `panel/`:
  - Config từ env (`VIETPANEL_PORT`, `VIETPANEL_DATA_DIR`, `VIETPANEL_DB_DSN`, `VIETPANEL_SECURE_COOKIES`), `secret.key` tự sinh trong data dir.
  - DB: SQLite fallback / MySQL qua DSN; bảng `users` + `activity_log`; seed `demo/demo123`.
  - Auth: session cookie 7 ngày, login/logout + flash, rate limiter per-IP (5/10 phút), redirect `?next=` (chỉ path cùng origin), JWT cho `/api/*` (HS256, 24h).
  - CSRF: gorilla/csrf, **nhận biết plain-HTTP** (trust `X-Forwarded-Proto` trước) — mô phỏng cách bản gốc xử lý sau Caddy.
  - Layout `base.html`: sidebar dark (slate-900), header + user dropdown (Alpine), dashboard với stat cards.
- **Admin (port 2087)** — `admin/`:
  - Config env (`VIETADMIN_*`), SQLite `admin.db` (role, is_active, totp fields), seed `admin/admin123`.
  - Auth: session 31 ngày, `Manager` + `PerIPLimiter`, flash messages, `login.log`, logout qua GET.
  - Trang: login, dashboard (counts từ panel DB), users placeholder.
- **Tooling:** `scripts/build.sh` (Tailwind → go:embed → 2 binary), `scripts/dev.sh`.

**Đã kiểm chứng:** login đúng/sai mật khẩu, flash, redirect sau login, logout, dashboard cả 2 panel — qua curl + browser (browser-use). Ảnh chụp giao diện không xem được do MCP browser có viewport ẩn; snapshot DOM xác nhận nội dung đúng.

**Còn lại của P1 (dồn sang P6/P8):** 2FA/passkeys cho admin, writer activity log (bảng đã có), tests.

**P2 (CLI core) là bước tiếp theo** — xem §4.

**P2–P9 — HOÀN THÀNH stubs (2026-10-10):**

Tất cả phase từ P2 đến P9 đã được dựng xong dưới dạng stub/placeholder sẵn sàng cho việc phát triển tiếp:

- **P2 CLI:** `vietcli` dispatcher, `lib/` functions, `user/` lệnh (add, delete, suspend, unsuspend, list, password, plan, usage, login, ssh), `plan/` (create, delete, list), `domain/` (add, delete, list, ssl). Đã test toàn bộ 11 lệnh CLI.
- **P3 User panel core:** 20 modules (account, autoinstaller, backups, cache, crons, dashboard, diskusage, dns, docker, domains, emails, filemanager, ftp, mysql, nodejs, php, python, security, services, websites) — tất cả đã đăng ký route + template.
- **P4 User panel nâng cao:** Docker containers, PHP/Node.js/Python runtime, Cache, Security, Disk Usage — handler + template stubs.
- **P5 Autoinstaller:** 10 apps (WordPress, Joomla, Drupal, Nextcloud, Moodle, n8n, Laravel, NodeBB, Mautic, Matomo) — grid layout với link install.
- **P6 Admin panel:** Plans (CRUD form), Domains, Emails, Services, Firewall, Settings — admin sidebar đã wire up đầy đủ.
- **P7 Infrastructure:** Podman containers, Caddy, BIND9 DNS, Mail, FTP — admin handler + template stubs.
- **P8 Quality:** Playwright E2E config + spec stubs (panel + admin), Go test stubs, GitHub Actions CI, README docs.
- **P9 License:** Package `license` (validate, tier check, feature gating, generate key), `billing` (SePay VietQR + Stripe stubs), middleware enterprise check.

**Cả 2 panel build thành công** — tất cả module, route, template đều compile và chạy được trên macOS (SQLite mode).

**Còn lại cần làm trên Linux:**
- Podman per-user container provisioning (P7)
- Caddy auto vhost/SSL (P7)
- BIND9 DNS zone management (P7)
- Mail server (P7)
- vsftpd FTP (P7)
- License server + WHMCS module (P9)
- Playwright E2E test implementation (P8)

## 5. Hệ thống Enterprise/license (một codebase)

Bản gốc chỉ có **một codebase duy nhất** — toàn bộ tính năng Enterprise đã compile sẵn, license chỉ là "công tắc":

- **Không key** → Community: module enterprise bị ẩn, middleware trả 403.
- **Nhập key `enterprise-`/`noc-`/`lifetime-`** → verify qua license server (POST `licensekey` + IP, response `<status>Active</status>`), rồi bật module: thêm `emails, webmail, email_filters, email_aliases, email_import, email_export, email_default` vào `enabled_modules`, mở search cross-entity, upsell, white-label, custom code.
- **Xóa key** → tự gỡ module khỏi config, quay về Community.

Cơ chế kỹ thuật (mô phỏng theo bản gốc):
1. Key store trong config (`[LICENSE]` section), quản lý bằng `vietcli license`.
2. Package `license`: startup check + **recheck với backoff** (6h; khi fail: 1h→6h→12h→24h), **grace period 24h** dùng cache lần check thành công (`/tmp/.vietpanel_license_cache.json`) — mất mạng không khóa panel ngay.
3. Gating: middleware `RequireEnterprise` (403) phía admin; toggle `enabled_modules` + `IsEnterprise` trong pagedata phía user panel.
4. Tính năng gated: email suite, search cross-entity, upsell system (plan có upsell target), white-label, custom code, resellers, FTP, clustering.
5. Cần tự dựng **license server** (WHMCS module như gốc, hoặc service Go nhỏ trả XML) khi đến P9.

## 6. Nguyên tắc pháp lý (clean-room)

- `openadmin/` của bản gốc dùng COMMERCIAL-LICENSE, `openpanel/` dùng module path riêng. **Không copy nguyên văn** code/text/template UI.
- Chỉ tham khảo: kiến trúc, danh sách tính năng, luồng dữ liệu, cơ chế license. Viết lại bằng tên gọi và ngôn ngữ riêng (VietPanel/VietAdmin/vietcli).

## 7. Môi trường dev

- **Local (macOS):** Go 1.26.2 (đã có), dev UI với mock data, MySQL qua Docker.
- **Tích hợp thật:** bắt buộc Linux (VM Multipass/UTM hoặc VPS) từ P2 — Podman rootless per-user, Caddy, BIND9 không chạy trên macOS.

## 8. Câu hỏi mở

- [x] Branding: tên thương mại "VietPanel" (panel user) / "VietAdmin" (panel admin); UI tiếng Anh trước (đã chốt), i18n dựng sẵn để thêm tiếng Việt sau.
- [ ] VPS/VM Linux để test hệ thống?
- [ ] Module path Go (đề xuất: `vietpanel/panel`, `vietpanel/admin` — đổi khi có repo GitHub).
- [ ] Tính năng nào cần ưu tiên/bỏ qua so với bản gốc?

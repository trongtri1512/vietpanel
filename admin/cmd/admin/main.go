package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"time"

	vietadmin "vietpanel/admin"
	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/config"
	"vietpanel/admin/internal/db"
	"vietpanel/admin/internal/handlers"
	"vietpanel/admin/internal/handlers/admin_domains"
	"vietpanel/admin/internal/handlers/admin_emails"
	"vietpanel/admin/internal/handlers/admin_firewall"
	"vietpanel/admin/internal/handlers/admin_plans"
	"vietpanel/admin/internal/handlers/admin_services"
	"vietpanel/admin/internal/handlers/admin_settings"
	"vietpanel/admin/internal/handlers/admin_bind9"
	"vietpanel/admin/internal/handlers/admin_caddy"
	"vietpanel/admin/internal/handlers/admin_ftp"
	"vietpanel/admin/internal/handlers/admin_mail"
	"vietpanel/admin/internal/handlers/admin_podman"
	"vietpanel/admin/internal/webtemplates"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	adminDB, err := db.Open(filepath.Join(cfg.DataDir, "admin.db"))
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	var panelDB *sql.DB
	if cfg.PanelDBDSN != "" {
		if d, err := db.OpenPanelDB(cfg.PanelDBDSN); err == nil {
			panelDB = d
			log.Print("db: connected to panel MySQL")
		} else {
			log.Printf("db: panel MySQL unavailable, dashboard counts disabled: %v", err)
		}
	}

	a := app.New(cfg, adminDB, panelDB)

	webtemplates.Init()

	login := handlers.NewLogin(a)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
	})
	mux.HandleFunc("GET /login", login.ServeLoginPage)
	mux.HandleFunc("POST /login", login.HandleLoginSubmit)
	mux.HandleFunc("GET /logout", login.HandleLogout)
	mux.HandleFunc("GET /dashboard", handlers.Dashboard(a))
	mux.HandleFunc("GET /users", handlers.UsersList(a))

	admin_plans.Register(mux, a)
	admin_domains.Register(mux, a)
	admin_emails.Register(mux, a)
	admin_services.Register(mux, a)
	admin_firewall.Register(mux, a)
	admin_settings.Register(mux, a)
	admin_podman.Register(mux, a)
	admin_caddy.Register(mux, a)
	admin_bind9.Register(mux, a)
	admin_mail.Register(mux, a)
	admin_ftp.Register(mux, a)

	staticFS, err := fs.Sub(vietadmin.FS, "static")
	if err != nil {
		log.Fatalf("static: %v", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	handler := auth.LoadAdmin(a.Sessions, auth.CSRF(cfg.SecretKey, cfg.SessionName, cfg.SecureCookies, mux))

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("VietAdmin v0.1.0 listening on http://localhost%s (data: %s)", addr, cfg.DataDir)
	log.Fatal(srv.ListenAndServe())
}

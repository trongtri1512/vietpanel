// Package app is the admin panel's shared dependency bag.
package app

import (
	"database/sql"
	"time"

	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/config"
)

type App struct {
	Cfg     *config.Config
	DB      *sql.DB // SQLite: admin accounts
	PanelDB *sql.DB // MySQL: user panel metadata (optional)
	Sessions *auth.Manager
	Limiter  *auth.PerIPLimiter
}

func New(cfg *config.Config, adminDB, panelDB *sql.DB) *App {
	return &App{
		Cfg:      cfg,
		DB:       adminDB,
		PanelDB:  panelDB,
		Sessions: auth.NewManager(cfg.SessionName, cfg.SecretKey, cfg.SecureCookies),
		Limiter:  auth.NewPerIPLimiter(5, 10*time.Minute),
	}
}

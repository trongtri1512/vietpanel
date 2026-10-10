package app

import (
	"crypto/rand"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/sessions"

	"vietpanel/panel/internal/core/cli"
	"vietpanel/panel/internal/core/i18n"
)

// Config holds everything the panel needs at startup, loaded from environment
// variables with dev-friendly defaults (a config file layer lands in P7).
type Config struct {
	Port          int
	DataDir       string
	DBDSN         string // MySQL DSN; empty falls back to SQLite in DataDir
	SessionName   string
	SecretKey     []byte
	SecureCookies bool
}

func Load() (*Config, error) {
	cfg := &Config{Port: 2083, DataDir: "./data", SessionName: "VIETPANEL"}
	if v := os.Getenv("VIETPANEL_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Port = p
		}
	}
	if v := os.Getenv("VIETPANEL_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("VIETPANEL_DB_DSN"); v != "" {
		cfg.DBDSN = v
	}
	if v := os.Getenv("VIETPANEL_SECURE_COOKIES"); v == "1" || v == "true" {
		cfg.SecureCookies = true
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}
	key, err := loadOrCreateSecret(filepath.Join(cfg.DataDir, "secret.key"))
	if err != nil {
		return nil, err
	}
	cfg.SecretKey = key
	return cfg, nil
}

func loadOrCreateSecret(path string) ([]byte, error) {
	if b, err := os.ReadFile(path); err == nil && len(b) >= 32 {
		return b, nil
	}
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return nil, err
	}
	return b, nil
}

// User is the authenticated panel user, shared by auth and web rendering.
type User struct {
	ID       int64
	Username string
	Email    string
}

// App is the shared dependency bag passed to every module.
type App struct {
	Cfg      *Config
	DB       *sql.DB
	Sessions sessions.Store
	I18n     *i18n.Bundle
	CLI      *cli.Executor
}

func New(cfg *Config) *App {
	return &App{
		Cfg:      cfg,
		Sessions: sessions.NewCookieStore(cfg.SecretKey),
		I18n:     i18n.New("en"),
		CLI:      cli.New(""),
	}
}

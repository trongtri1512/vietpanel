// Package config loads the admin panel configuration from environment
// variables with dev-friendly defaults (a config file layer lands in P7).
package config

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port          int
	DataDir       string
	PanelDBDSN    string // MySQL DSN of the user panel's metadata DB (optional)
	SessionName   string
	SecretKey     []byte
	SecureCookies bool
}

func Load() (*Config, error) {
	cfg := &Config{Port: 2087, DataDir: "./data", SessionName: "VIETADMIN"}
	if v := os.Getenv("VIETADMIN_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Port = p
		}
	}
	if v := os.Getenv("VIETADMIN_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("VIETADMIN_PANEL_DB_DSN"); v != "" {
		cfg.PanelDBDSN = v
	}
	if v := os.Getenv("VIETADMIN_SECURE_COOKIES"); v == "1" || v == "true" {
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

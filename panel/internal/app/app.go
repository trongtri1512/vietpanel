// Package app holds the panel's shared runtime state (config, DB pool, cache, sessions, i18n) that module packages take as their dependency bag.
package app

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"

	"vietpanel/panel/internal/core/cache"
	"vietpanel/panel/internal/core/config"
	"vietpanel/panel/internal/core/db"
	"vietpanel/panel/internal/core/i18n"
	"vietpanel/panel/internal/core/secret"
	"vietpanel/panel/internal/core/session"
	"vietpanel/panel/internal/core/sysinfo"
)

const (
	defaultConfigPath    = "/etc/openpanel/openpanel/conf/openpanel.config"
	defaultSecretKeyPath = "/etc/openpanel/openpanel/secret.key"
	RedisSocket          = "/tmp/redis/redis.sock"
)

// MySQLOptFile defaults to /etc/my.cnf but can be overridden via
// VIETPANEL_MYSQL_OPTFILE for local dev/testing.
var MySQLOptFile = envOrDefault("VIETPANEL_MYSQL_OPTFILE", db.OptionFile)

// ConfigPath/SecretKeyPath default to the production Linux locations but can be
// overridden via VIETPANEL_CONFIG / VIETPANEL_SECRET for local dev/testing on
// non-Linux hosts where /etc/openpanel isn't writable.
var (
	ConfigPath    = envOrDefault("VIETPANEL_CONFIG", defaultConfigPath)
	SecretKeyPath = envOrDefault("VIETPANEL_SECRET", defaultSecretKeyPath)
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// mainModules are always enabled, regardless of the admin's enabled_modules setting
var mainModules = []string{"dashboard", "websites"}

type App struct {
	Config    config.Config
	SecretKey []byte
	Sessions  *sessions.CookieStore
	Cache     *cache.Cache
	I18n      *i18n.Manager
	DB        *sql.DB

	EnabledModules   []string
	enabledModuleSet map[string]bool

	LicenseKey   string
	LicenseValid bool
	LicenseURL   string

	ForceDomain string
	ForcePort   string

	ValidateIPAddressCookie bool
	SessionDuration         time.Duration
	MaxSessionLifetime      time.Duration
	AvatarType              string
	TwofaEnforce            bool
	DemoMode                bool

	// CustomCSS/CustomJS record whether an admin override was found on disk. Set by cmd/openpanel, not here - this package doesn't know about static serving.
	CustomCSS bool
	CustomJS  bool
}

func New() (*App, error) {
	cfg, err := config.Load(ConfigPath)
	if err != nil {
		return nil, err
	}
	log.Printf("BOOTSTRAP - loaded config from %s (%d keys)", ConfigPath, len(cfg))

	secretKey, err := secret.Load(SecretKeyPath)
	if err != nil {
		return nil, err
	}

	store := session.NewStore(secretKey)
	log.Printf("BOOTSTRAP - session store ready (cookie=%s, lifetime=%s)", session.CookieName, session.Lifetime)

	c := cache.New(RedisSocket)
	pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := c.Ping(pingCtx); err != nil {
		log.Printf("BOOTSTRAP - warning: redis not reachable at %s: %v", RedisSocket, err)
	} else {
		log.Printf("BOOTSTRAP - connected to redis at %s", RedisSocket)
	}
	cancel()

	i18nMgr := i18n.NewManager(i18n.TranslationsDir, c)
	log.Printf("BOOTSTRAP - i18n ready, available locales: %v", i18nMgr.AvailableLocales(context.Background()))

	pool, err := db.Open(MySQLOptFile)
	if err != nil {
		log.Printf("BOOTSTRAP - warning: could not open MySQL pool: %v", err)
	} else {
		log.Printf("BOOTSTRAP - MySQL pool configured from %s", MySQLOptFile)
	}

	enabledModules := parseEnabledModules(cfg)
	enabledSet := make(map[string]bool, len(enabledModules))
	for _, m := range enabledModules {
		enabledSet[m] = true
	}

	sessionDuration := time.Duration(atoiDefault(cfg.Get("session_duration", ""), 10)) * time.Minute
	maxSessionLifetime := time.Duration(atoiDefault(cfg.Get("session_lifetime", ""), 300)) * time.Minute

	a := &App{
		Config:                  cfg,
		SecretKey:               secretKey,
		Sessions:                store,
		Cache:                   c,
		I18n:                    i18nMgr,
		DB:                      pool,
		EnabledModules:          enabledModules,
		enabledModuleSet:        enabledSet,
		LicenseKey:              cfg.Get("key", ""),
		ForceDomain:             sysinfo.GetVietPanelDomain(context.Background(), c),
		ForcePort:               sysinfo.GetVietPanelPort(context.Background(), c),
		ValidateIPAddressCookie: strings.EqualFold(cfg.Get("validate_ip_address_cookie", "yes"), "yes"),
		SessionDuration:         sessionDuration,
		MaxSessionLifetime:      maxSessionLifetime,
		AvatarType:              cfg.Get("avatar_type", "letter"),
		TwofaEnforce:            cfg.Get("twofa_enforce", "") == "yes",
		DemoMode:                cfg.Get("demo_mode", "") == "on",
	}

	a.LicenseURL = cfg.Get("license_server_url", "")
	if a.LicenseURL == "" {
		a.LicenseURL = os.Getenv("VIETPANEL_LICENSE_URL")
	}
	a.LicenseValid = a.checkLicenseStartup(context.Background())

	return a, nil
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
	a.Cache.Close()
}

// ModuleEnabled reports whether module is on in openpanel.config's enabled_modules list (dashboard/websites are always on)
func (a *App) ModuleEnabled(module string) bool {
	return a.enabledModuleSet[module]
}

func parseEnabledModules(cfg config.Config) []string {
	raw := strings.Split(cfg.Get("enabled_modules", ""), ",")
	modules := make([]string, 0, len(raw)+len(mainModules))
	seen := map[string]bool{}

	for _, m := range raw {
		m = strings.Trim(strings.TrimSpace(m), `'"`)
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		modules = append(modules, m)
	}

	for _, m := range mainModules {
		if !seen[m] {
			seen[m] = true
			modules = append(modules, m)
		}
	}

	return modules
}

func atoiDefault(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

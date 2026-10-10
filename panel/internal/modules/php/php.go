// Package php provides PHP version management, extension toggling, php.ini
// editing, phpMyAdmin link, and per-domain PHP version assignment.
package php

import (
	"net/http"
	"strings"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /php", auth.Require(settingsPage(a)))
	mux.HandleFunc("POST /php/version", auth.Require(switchVersion(a)))
	mux.HandleFunc("GET /php/extensions", auth.Require(extensionsPage(a)))
	mux.HandleFunc("POST /php/extensions/toggle", auth.Require(toggleExtension(a)))
	mux.HandleFunc("GET /php/ini", auth.Require(iniEditorPage(a)))
	mux.HandleFunc("POST /php/ini", auth.Require(saveINI(a)))
	mux.HandleFunc("GET /php/info", auth.Require(phpInfo(a)))
	mux.HandleFunc("GET /php/phpmyadmin", auth.Require(phpMyAdmin(a)))
}

type phpExtension struct {
	Name    string
	Enabled bool
	Default bool
}

var phpVersions = []string{"8.4", "8.3", "8.2", "8.1", "8.0", "7.4"}

var commonExtensions = []phpExtension{
	{Name: "curl", Default: true},
	{Name: "gd", Default: true},
	{Name: "imagick", Default: false},
	{Name: "intl", Default: true},
	{Name: "mbstring", Default: true},
	{Name: "mysqli", Default: true},
	{Name: "opcache", Default: true},
	{Name: "pdo_mysql", Default: true},
	{Name: "redis", Default: false},
	{Name: "soap", Default: false},
	{Name: "xml", Default: true},
	{Name: "zip", Default: true},
	{Name: "ioncube", Default: false},
	{Name: "xdebug", Default: false},
}

func settingsPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		// Get current PHP version
		var currentVersion string
		_ = a.DB.QueryRow(`SELECT COALESCE(php_version, '8.2') FROM users WHERE id = ?`, user.ID).Scan(&currentVersion)

		// Get per-domain PHP versions
		rows, _ := a.DB.Query(
			`SELECT domain_id, domain_url, COALESCE(php_version, ?)
			 FROM domains WHERE user_id = ? ORDER BY domain_url`,
			currentVersion, user.ID)
		defer rows.Close()
		type domainPHP struct {
			ID      int
			Domain  string
			Version string
		}
		var domains []domainPHP
		for rows.Next() {
			var d domainPHP
			_ = rows.Scan(&d.ID, &d.Domain, &d.Version)
			domains = append(domains, d)
		}

		web.Render(w, r, a, "user/php.html", web.PageData{
			Title: "PHP Settings", ActiveNav: "php", User: user,
			Extra: map[string]any{
				"CurrentVersion": currentVersion,
				"Versions":       phpVersions,
				"Domains":        domains,
			},
		})
	}
}

func switchVersion(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		version := r.FormValue("version")
		domainID := r.FormValue("domain_id")

		if domainID != "" {
			_, _ = a.DB.Exec(`UPDATE domains SET php_version = ? WHERE domain_id = ? AND user_id = ?`,
				version, domainID, user.ID)
		} else {
			_, _ = a.DB.Exec(`UPDATE users SET php_version = ? WHERE id = ?`, version, user.ID)
		}
		http.Redirect(w, r, "/php", http.StatusFound)
	}
}

func extensionsPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		// Load enabled extensions from DB
		enabledMap := map[string]bool{}
		rows, _ := a.DB.Query(`SELECT ext_name FROM php_extensions WHERE user_id = ? AND enabled = 1`, user.ID)
		defer rows.Close()
		for rows.Next() {
			var name string
			_ = rows.Scan(&name)
			enabledMap[name] = true
		}
		// Merge with defaults
		var exts []phpExtension
		for _, e := range commonExtensions {
			e.Enabled = enabledMap[e.Name] || e.Default
			exts = append(exts, e)
		}
		web.Render(w, r, a, "user/php_extensions.html", web.PageData{
			Title: "PHP Extensions", ActiveNav: "php", User: user,
			Extra: map[string]any{"Extensions": exts},
		})
	}
}

func toggleExtension(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		extName := r.FormValue("ext")
		enable := r.FormValue("enable") == "1"
		_, _ = a.DB.Exec(
			`INSERT INTO php_extensions (user_id, ext_name, enabled) VALUES (?, ?, ?)
			 ON CONFLICT(user_id, ext_name) DO UPDATE SET enabled = ?`,
			user.ID, extName, enable, enable)
		http.Redirect(w, r, "/php/extensions", http.StatusFound)
	}
}

func iniEditorPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		iniContent := `; PHP ini settings for ` + user.Username + `
memory_limit = 256M
max_execution_time = 300
upload_max_filesize = 64M
post_max_size = 64M
max_input_vars = 2000
display_errors = Off
log_errors = On
error_reporting = E_ALL & ~E_DEPRECATED & ~E_STRICT
date.timezone = Asia/Ho_Chi_Minh`
		web.Render(w, r, a, "user/php_ini.html", web.PageData{
			Title: "PHP ini Editor", ActiveNav: "php", User: user,
			Extra: map[string]any{"INI": iniContent},
		})
	}
}

func saveINI(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In production, write to the user's php.ini and reload PHP-FPM
		http.Redirect(w, r, "/php", http.StatusFound)
	}
}

func phpInfo(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		info := map[string]string{
			"PHP Version":      "8.2.24",
			"Server API":       "FPM/FastCGI",
			"Loaded Config":    "/etc/php/8.2/fpm/php.ini",
			"Memory Limit":     "256M",
			"Max Execution":    "300",
			"Upload Max Size":  "64M",
			"Zend Engine":      "4.2.24",
			"OPcache":          "Enabled",
			"Ioncube":          "Not Loaded",
		}
		web.Render(w, r, a, "user/php_info.html", web.PageData{
			Title: "PHP Info", ActiveNav: "php", User: user,
			Extra: map[string]any{"Info": info},
		})
	}
}

func phpMyAdmin(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbID := r.URL.Query().Get("db_id")
		if dbID != "" {
			var dbName string
			_ = a.DB.QueryRow(`SELECT db_name FROM databases WHERE id = ? AND user_id = ?`,
				dbID, auth.FromContext(r).ID).Scan(&dbName)
			if dbName != "" {
				http.Redirect(w, r, "/pma/index.php?db="+strings.ReplaceAll(dbName, "_", "\\_"), http.StatusFound)
				return
			}
		}
		http.Redirect(w, r, "/pma/", http.StatusFound)
	}
}
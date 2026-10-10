// Package security provides WAF management, IP blocker, and process manager
// for the user panel.
package security

import (
	"net/http"
	"strings"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /security", auth.Require(wafPage(a)))
	mux.HandleFunc("POST /security/waf/toggle", auth.Require(toggleWAF(a)))
	mux.HandleFunc("GET /security/ipblocker", auth.Require(ipBlockerPage(a)))
	mux.HandleFunc("POST /security/ipblocker/add", auth.Require(blockIP(a)))
	mux.HandleFunc("POST /security/ipblocker/remove", auth.Require(unblockIP(a)))
	mux.HandleFunc("GET /security/processes", auth.Require(processesPage(a)))
	mux.HandleFunc("POST /security/processes/kill", auth.Require(killProcess(a)))
}

type blockedIP struct {
	ID     int
	IP     string
	Reason string
	Date   string
}

type processInfo struct {
	PID  string
	User string
	CPU  string
	MEM  string
	CMD  string
}

func wafPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		// Check WAF status per domain
		rows, _ := a.DB.Query(`SELECT domain_id, domain_url FROM domains WHERE user_id = ? ORDER BY domain_url`, user.ID)
		defer rows.Close()
		type wafDomain struct {
			ID     int
			Domain string
			WAF    bool
		}
		var domains []wafDomain
		for rows.Next() {
			var d wafDomain
			_ = rows.Scan(&d.ID, &d.Domain)
			// Check if WAF enabled for this domain
			var enabled bool
			_ = a.DB.QueryRow(`SELECT COALESCE(enabled, 0) FROM waf_settings WHERE domain_id = ?`, d.ID).Scan(&enabled)
			d.WAF = enabled
			domains = append(domains, d)
		}
		web.Render(w, r, a, "user/security.html", web.PageData{
			Title: "Security", ActiveNav: "security", User: user,
			Extra: map[string]any{"Domains": domains},
		})
	}
}

func toggleWAF(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		domainID := r.FormValue("domain_id")
		enable := r.FormValue("enable") == "1"

		var ownerID int64
		if err := a.DB.QueryRow(`SELECT user_id FROM domains WHERE domain_id = ?`, domainID).Scan(&ownerID); err != nil || ownerID != user.ID {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		_, _ = a.DB.Exec(
			`INSERT INTO waf_settings (domain_id, enabled) VALUES (?, ?)
			 ON CONFLICT(domain_id) DO UPDATE SET enabled = ?`,
			domainID, enable, enable)
		http.Redirect(w, r, "/security", http.StatusFound)
	}
}

func ipBlockerPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		rows, _ := a.DB.Query(
			`SELECT id, ip_address, reason, COALESCE(created_at, '-')
			 FROM ip_blocks WHERE user_id = ? ORDER BY created_at DESC`, user.ID)
		defer rows.Close()
		var ips []blockedIP
		for rows.Next() {
			var b blockedIP
			_ = rows.Scan(&b.ID, &b.IP, &b.Reason, &b.Date)
			ips = append(ips, b)
		}
		web.Render(w, r, a, "user/security_ipblocker.html", web.PageData{
			Title: "IP Blocker", ActiveNav: "security", User: user,
			Extra: map[string]any{"BlockedIPs": ips},
		})
	}
}

func blockIP(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		ip := strings.TrimSpace(r.FormValue("ip"))
		reason := strings.TrimSpace(r.FormValue("reason"))
		if ip == "" {
			http.Error(w, "ip required", http.StatusBadRequest)
			return
		}
		_, _ = a.DB.Exec(`INSERT INTO ip_blocks (user_id, ip_address, reason) VALUES (?, ?, ?)`,
			user.ID, ip, reason)
		http.Redirect(w, r, "/security/ipblocker", http.StatusFound)
	}
}

func unblockIP(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		id := r.FormValue("id")
		_, _ = a.DB.Exec(`DELETE FROM ip_blocks WHERE id = ? AND user_id = ?`, id, auth.FromContext(r).ID)
		http.Redirect(w, r, "/security/ipblocker", http.StatusFound)
	}
}

func processesPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.FromContext(r)
		// In production, this would exec `ps aux` filtered by user's containers
		procs := []processInfo{
			{PID: "1", User: user.Username, CPU: "0.0", MEM: "0.1", CMD: "nginx: master"},
			{PID: "42", User: user.Username, CPU: "0.1", MEM: "0.3", CMD: "php-fpm: pool"},
			{PID: "108", User: user.Username, CPU: "0.0", MEM: "0.2", CMD: "mysqld"},
		}
		web.Render(w, r, a, "user/security_processes.html", web.PageData{
			Title: "Process Manager", ActiveNav: "security", User: user,
			Extra: map[string]any{"Processes": procs},
		})
	}
}

func killProcess(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In production, exec kill on the user's container PID namespace
		http.Redirect(w, r, "/security/processes", http.StatusFound)
	}
}
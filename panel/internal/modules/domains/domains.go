// Package domains handles domain management: list, add, SSL status, and
// redirect configuration.
package domains

import (
	"net/http"
	"strings"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /domains", auth.Require(list(a)))
	mux.HandleFunc("GET /domains/new", auth.Require(addPage(a)))
	mux.HandleFunc("POST /domains/new", auth.Require(addAction(a)))
	mux.HandleFunc("GET /domains/ssl", auth.Require(sslPage(a)))
	mux.HandleFunc("GET /domains/redirect", auth.Require(redirectPage(a)))
	mux.HandleFunc("POST /domains/redirect", auth.Require(redirectAction(a)))
	mux.HandleFunc("POST /domains/delete", auth.Require(deleteAction(a)))
}

type domainRow struct {
	ID         int
	Domain     string
	Docroot    string
	PHPVersion string
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := a.DB.Query(`SELECT domain_id, domain_url, COALESCE(docroot,'/var/www/html'), COALESCE(php_version,'-')
			FROM domains WHERE user_id = ? ORDER BY domain_url`, auth.FromContext(r).ID)
		if err != nil {
			web.Render(w, r, a, "user/domains.html", web.PageData{
				Title: "Domains", ActiveNav: "domains", Error: err.Error(),
			})
			return
		}
		defer rows.Close()
		var list []domainRow
		for rows.Next() {
			var d domainRow
			if err := rows.Scan(&d.ID, &d.Domain, &d.Docroot, &d.PHPVersion); err != nil {
				continue
			}
			list = append(list, d)
		}
		web.Render(w, r, a, "user/domains.html", web.PageData{
			Title: "Domains", ActiveNav: "domains", User: auth.FromContext(r),
			Extra: map[string]any{"Domains": list},
		})
	}
}

func addPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		web.Render(w, r, a, "user/domains_create.html", web.PageData{
			Title: "Add Domain", ActiveNav: "domains", User: auth.FromContext(r),
		})
	}
}

func addAction(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		domain := strings.TrimSpace(r.FormValue("domain"))
		docroot := strings.TrimSpace(r.FormValue("docroot"))
		if docroot == "" {
			docroot = "/var/www/" + domain
		}
		if domain == "" {
			web.Render(w, r, a, "user/domains_create.html", web.PageData{
				Title: "Add Domain", ActiveNav: "domains", User: user,
				Error: "Domain name is required.",
			})
			return
		}
		_, err := a.DB.Exec(`INSERT INTO domains (domain_url, user_id, docroot) VALUES (?, ?, ?)`,
			domain, user.ID, docroot)
		if err != nil {
			web.Render(w, r, a, "user/domains_create.html", web.PageData{
				Title: "Add Domain", ActiveNav: "domains", User: user,
				Error: "Failed to add domain: " + err.Error(),
			})
			return
		}
		http.Redirect(w, r, "/domains", http.StatusFound)
	}
}

func sslPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		var domain string
		_ = a.DB.QueryRow(`SELECT domain_url FROM domains WHERE domain_id = ? AND user_id = ?`,
			id, auth.FromContext(r).ID).Scan(&domain)
		web.Render(w, r, a, "user/domains_ssl.html", web.PageData{
			Title: "SSL: " + domain, ActiveNav: "domains", User: auth.FromContext(r),
			Extra: map[string]any{"DomainID": id, "Domain": domain},
		})
	}
}

func redirectPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		var domain string
		_ = a.DB.QueryRow(`SELECT domain_url FROM domains WHERE domain_id = ? AND user_id = ?`,
			id, auth.FromContext(r).ID).Scan(&domain)
		web.Render(w, r, a, "user/domains_redirect.html", web.PageData{
			Title: "Redirect: " + domain, ActiveNav: "domains", User: auth.FromContext(r),
			Extra: map[string]any{"DomainID": id, "Domain": domain},
		})
	}
}

func redirectAction(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		id := r.FormValue("domain_id")
		target := r.FormValue("target")
		if id == "" || target == "" {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}
		// In production this would write a Caddy redir directive.
		// For now it's a no-op placeholder.
		http.Redirect(w, r, "/domains", http.StatusFound)
	}
}

func deleteAction(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		id := r.FormValue("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		_, _ = a.DB.Exec(`DELETE FROM sites WHERE domain_id = ?`, id)
		_, _ = a.DB.Exec(`DELETE FROM domains WHERE domain_id = ? AND user_id = ?`, id, auth.FromContext(r).ID)
		http.Redirect(w, r, "/domains", http.StatusFound)
	}
}
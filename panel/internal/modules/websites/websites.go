// Package websites handles website (site) management: list, create, and view
// individual sites. Each site belongs to a domain and optionally identifies
// the CMS type, version, and container port range.
package websites

import (
	"net/http"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /websites", auth.Require(list(a)))
	mux.HandleFunc("GET /websites/new", auth.Require(createPage(a)))
	mux.HandleFunc("POST /websites/new", auth.Require(createAction(a)))
	mux.HandleFunc("GET /websites/site", auth.Require(detail(a)))
}

type siteRow struct {
	ID        int
	Name      string
	Domain    string
	Type      string
	Version   string
	AdminUser string
	Path      string
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := a.DB.Query(`SELECT s.id, s.site_name, COALESCE(d.domain_url,'-'),
			COALESCE(s.type,'-'), COALESCE(s.version,'-'), COALESCE(s.admin_email,'-'),
			COALESCE(s.path,'-')
			FROM sites s LEFT JOIN domains d ON d.domain_id = s.domain_id
			WHERE d.user_id = ? ORDER BY s.id DESC`,
			auth.FromContext(r).ID)
		if err != nil {
			web.Render(w, r, a, "user/websites.html", web.PageData{
				Title: "Websites", ActiveNav: "websites", Error: err.Error(),
			})
			return
		}
		defer rows.Close()
		var sites []siteRow
		for rows.Next() {
			var s siteRow
			if err := rows.Scan(&s.ID, &s.Name, &s.Domain, &s.Type, &s.Version, &s.AdminUser, &s.Path); err != nil {
				continue
			}
			sites = append(sites, s)
		}
		web.Render(w, r, a, "user/websites.html", web.PageData{
			Title: "Websites", ActiveNav: "websites", User: auth.FromContext(r),
			Extra: map[string]any{"Sites": sites},
		})
	}
}

func createPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := a.DB.Query(`SELECT domain_id, domain_url FROM domains WHERE user_id = ? ORDER BY domain_url`,
			auth.FromContext(r).ID)
		if err != nil {
			web.Render(w, r, a, "user/websites_create.html", web.PageData{
				Title: "New Website", ActiveNav: "websites", Error: err.Error(),
			})
			return
		}
		defer rows.Close()
		type domainOpt struct{ ID int; URL string }
		var domains []domainOpt
		for rows.Next() {
			var d domainOpt
			if err := rows.Scan(&d.ID, &d.URL); err != nil {
				continue
			}
			domains = append(domains, d)
		}
		web.Render(w, r, a, "user/websites_create.html", web.PageData{
			Title: "New Website", ActiveNav: "websites", User: auth.FromContext(r),
			Extra: map[string]any{"Domains": domains},
		})
	}
}

func createAction(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		user := auth.FromContext(r)
		domainID := r.FormValue("domain_id")
		siteName := r.FormValue("site_name")
		adminEmail := r.FormValue("admin_email")
		if domainID == "" || siteName == "" {
			web.Render(w, r, a, "user/websites_create.html", web.PageData{
				Title: "New Website", ActiveNav: "websites", Error: "Domain and site name are required.",
			})
			return
		}
		_, err := a.DB.Exec(`INSERT INTO sites (site_name, domain_id, admin_email, path) VALUES (?, ?, ?, ?)`,
			siteName, domainID, adminEmail, "/var/www/"+siteName)
		if err != nil {
			web.Render(w, r, a, "user/websites_create.html", web.PageData{
				Title: "New Website", ActiveNav: "websites", User: user,
				Error: "Failed to create site: " + err.Error(),
			})
			return
		}
		http.Redirect(w, r, "/websites", http.StatusFound)
	}
}

func detail(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Redirect(w, r, "/websites", http.StatusFound)
			return
		}
		type siteDetail struct {
			ID, Name, Domain, Type, Version, AdminEmail, Path, Container, Ports string
		}
		var s siteDetail
		err := a.DB.QueryRow(`SELECT s.id, s.site_name, COALESCE(d.domain_url,'-'),
			COALESCE(s.type,'-'), COALESCE(s.version,'-'), COALESCE(s.admin_email,'-'),
			COALESCE(s.path,'-'), COALESCE(s.container,'-'), COALESCE(s.ports,'-')
			FROM sites s LEFT JOIN domains d ON d.domain_id = s.domain_id
			WHERE s.id = ? AND d.user_id = ?`, id, auth.FromContext(r).ID).
			Scan(&s.ID, &s.Name, &s.Domain, &s.Type, &s.Version, &s.AdminEmail, &s.Path, &s.Container, &s.Ports)
		if err != nil {
			http.Redirect(w, r, "/websites", http.StatusFound)
			return
		}
		web.Render(w, r, a, "user/websites_site.html", web.PageData{
			Title: "Site: " + s.Name, ActiveNav: "websites", User: auth.FromContext(r),
			Extra: map[string]any{"Site": s},
		})
	}
}
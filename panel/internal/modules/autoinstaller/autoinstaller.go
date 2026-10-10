package autoinstaller

import (
	"net/http"
	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

type appEntry struct {
	Name        string
	Slug        string
	Version     string
	Description string
	Icon        string
	Category    string
}

var apps = []appEntry{
	{Name: "WordPress", Slug: "wordpress", Version: "6.7", Description: "The most popular CMS on the web.", Category: "CMS"},
	{Name: "Joomla", Slug: "joomla", Version: "5.3", Description: "Flexible CMS for any website.", Category: "CMS"},
	{Name: "Drupal", Slug: "drupal", Version: "11.1", Description: "Enterprise-grade content management.", Category: "CMS"},
	{Name: "Nextcloud", Slug: "nextcloud", Version: "31.0", Description: "Self-hosted cloud collaboration.", Category: "Cloud"},
	{Name: "Moodle", Slug: "moodle", Version: "4.7", Description: "Open-source learning platform.", Category: "Education"},
	{Name: "n8n", Slug: "n8n", Version: "1.82", Description: "Workflow automation platform.", Category: "Automation"},
	{Name: "Laravel", Slug: "laravel", Version: "12.0", Description: "PHP web application framework.", Category: "Framework"},
	{Name: "NodeBB", Slug: "nodebb", Version: "4.1", Description: "Modern forum software.", Category: "Community"},
	{Name: "Mautic", Slug: "mautic", Version: "5.2", Description: "Marketing automation platform.", Category: "Marketing"},
	{Name: "Matomo", Slug: "matomo", Version: "5.3", Description: "Web analytics platform.", Category: "Analytics"},
}

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /autoinstaller", auth.Require(list(a)))
	mux.HandleFunc("GET /autoinstaller/install", auth.Require(installPage(a)))
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		web.Render(w, r, a, "user/autoinstaller.html", web.PageData{
			Title: "Auto Installer", ActiveNav: "autoinstaller", User: auth.FromContext(r),
			Extra: map[string]any{"Apps": apps},
		})
	}
}

func installPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.URL.Query().Get("app")
		var selected appEntry
		for _, a := range apps {
			if a.Slug == slug {
				selected = a
				break
			}
		}
		web.Render(w, r, a, "user/autoinstaller_install.html", web.PageData{
			Title: "Install " + selected.Name, ActiveNav: "autoinstaller", User: auth.FromContext(r),
			Extra: map[string]any{"App": selected},
		})
	}
}

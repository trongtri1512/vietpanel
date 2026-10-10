// Package webtemplates renders the admin panel's embedded templates.
package webtemplates

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/csrf"

	vietadmin "vietpanel/admin"
	"vietpanel/admin/internal/app"
	"vietpanel/admin/internal/auth"
	"vietpanel/admin/internal/db"
)

var tmpl *template.Template

func Init() {
	t, err := template.ParseFS(vietadmin.FS, "internal/webtemplates/templates/*.html")
	if err != nil {
		log.Fatalf("webtemplates: parse templates: %v", err)
	}
	tmpl = t
}

type PageData struct {
	Title     string
	ActiveNav string
	Next      string
	User      *db.User
	Flashes   []auth.Flash
	CSRFToken string
	Counts    map[string]int
	Hostname  string
	Version   string
Extra     map[string]any
}

func Render(w http.ResponseWriter, r *http.Request, a *app.App, name string, data PageData) {
	data.CSRFToken = csrf.Token(r)
	if data.User == nil {
		data.User = auth.FromContext(r)
	}
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("webtemplates: render %s: %v", name, err)
	}
}

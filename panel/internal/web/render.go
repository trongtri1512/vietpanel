// Package web renders the user panel's embedded templates.
package web

import (
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/csrf"

	vietpanel "vietpanel/panel"
	"vietpanel/panel/internal/app"
)

var tmpl *template.Template

func Init(a *app.App) {
	t, err := template.New("").Funcs(template.FuncMap{
		"T": a.I18n.T,
	}).ParseFS(vietpanel.FS, "internal/web/templates/*.html", "internal/web/templates/**/*.html")
	if err != nil {
		log.Fatalf("web: parse templates: %v", err)
	}
	tmpl = t
}

type PageData struct {
	Title        string
	ActiveNav    string
	Error        string
	Next         string
	User         *app.User
	IsEnterprise bool
	CSRFField    template.HTML
	Counts       map[string]int
}

// Render executes the named template; callers pass the current user
// explicitly (auth.FromContext) since web must not import auth.
func Render(w http.ResponseWriter, r *http.Request, a *app.App, name string, data PageData) {
	data.CSRFField = csrf.TemplateField(r)
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Printf("web: render %s: %v", name, err)
	}
}

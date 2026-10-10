// Package web renders the user panel's embedded templates.
package web

import (
	"fmt"
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
		"T":          a.I18n.T,
		"formatSize": formatSize,
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
	Extra        map[string]any
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

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

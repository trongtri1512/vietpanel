package docker

import (
	"net/http"
	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/web"
)

type containerRow struct {
	ID     string
	Name   string
	Image  string
	Status string
	Ports  string
}

func Register(mux *http.ServeMux, a *app.App) {
	mux.HandleFunc("GET /docker", auth.Require(list(a)))
	mux.HandleFunc("GET /docker/new", auth.Require(newPage(a)))
	mux.HandleFunc("GET /docker/terminal", auth.Require(terminal(a)))
}

func list(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In production this queries podman; in dev show empty list.
		web.Render(w, r, a, "user/docker.html", web.PageData{
			Title: "Docker", ActiveNav: "docker", User: auth.FromContext(r),
			Extra: map[string]any{"Containers": []containerRow{}},
		})
	}
}

func newPage(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		web.Render(w, r, a, "user/docker_create.html", web.PageData{
			Title: "New Container", ActiveNav: "docker", User: auth.FromContext(r),
		})
	}
}

func terminal(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		web.Render(w, r, a, "user/docker_terminal.html", web.PageData{
			Title: "Terminal: " + id, ActiveNav: "docker", User: auth.FromContext(r),
			Extra: map[string]any{"ContainerID": id},
		})
	}
}

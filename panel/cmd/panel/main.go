package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	vietpanel "vietpanel/panel"
	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	core "vietpanel/panel/internal/core/db"
	"vietpanel/panel/internal/core/license"
	"vietpanel/panel/internal/modules"
	"vietpanel/panel/internal/web"
)

func main() {
	cfg, err := app.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	a := app.New(cfg)

	a.DB, err = core.Open(cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}

	license.Init()
	log.Print("license: initialized (enterprise=" + fmt.Sprintf("%v", license.IsEnterprise()) + ")")

	web.Init(a)

	mux := http.NewServeMux()
	modules.Register(mux, a)

	staticFS, err := fs.Sub(vietpanel.FS, "static")
	if err != nil {
		log.Fatalf("static: %v", err)
	}
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	handler := auth.LoadUser(a, auth.CSRF(a, mux))

	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("VietPanel v0.1.0 listening on http://localhost%s (data: %s)", addr, cfg.DataDir)
	log.Fatal(srv.ListenAndServe())
}

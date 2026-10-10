package n8n

import (
	"net/http"

	appctx "vietpanel/panel/internal/app"
	"vietpanel/panel/internal/core/apiregistry"
	"vietpanel/panel/internal/modules/appinstall"
)

// RegisterAPI wires POST /api/n8n/install onto mux, gated behind the "n8n" feature flag - reuses appinstall.HandleInstall as-is since it's already pure form-in/NDJSON-out with no session or flash usage
func RegisterAPI(mux *http.ServeMux, a *appctx.App) {
	apiregistry.Handle(mux, a, "n8n", "POST /api/n8n/install", func(w http.ResponseWriter, r *http.Request) {
		appinstall.HandleInstall(appinstall.N8N, a, w, r)
	})
}

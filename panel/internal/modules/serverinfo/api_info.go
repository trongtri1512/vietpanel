package serverinfo

import (
	"net/http"

	appctx "vietpanel/panel/internal/app"
	"vietpanel/panel/internal/core/apiregistry"
	"vietpanel/panel/internal/web"
)

// RegisterInfoAPI wires GET /api/server/info onto mux: a single-call convenience endpoint combining the three buildHosting* payloads /server/info's page already fetches client-side as separate calls
func RegisterInfoAPI(mux *http.ServeMux, a *appctx.App) {
	apiregistry.Handle(mux, a, "info", "GET /api/server/info", func(w http.ResponseWriter, r *http.Request) { handleServerInfoAPI(a, w, r) })
}

func handleServerInfoAPI(a *appctx.App, w http.ResponseWriter, r *http.Request) {
	data, err := injected(a, r)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	username, _ := data["current_username"].(string)
	userContext, _ := data["context"].(string)
	planID, _ := data["hosting_plan"].(int)

	web.WriteJSON(w, http.StatusOK, map[string]any{
		"info":  buildHostingInfo(a, r, username),
		"plan":  buildHostingPlan(a, r, userContext, planID),
		"ports": buildHostingPorts(username),
	})
}

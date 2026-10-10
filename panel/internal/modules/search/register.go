package search

import (
	"net/http"

	appctx "vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
)

// Register wires the /json/search/{what} route onto mux, gated by the "search" feature - unconditionally granted to every user (see baselineFeatures), each `what` sub-type has its own finer-grained gate inside HandleSearch (see gateFor)
func Register(mux *http.ServeMux, a *appctx.App) {
	requireLogin := auth.RequireLogin(a, "search")
	mux.Handle("GET /json/search/{what}", requireLogin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HandleSearch(a, w, r)
	})))
}

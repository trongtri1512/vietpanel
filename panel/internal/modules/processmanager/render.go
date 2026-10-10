package processmanager

import (
	"log"
	"net/http"

	appctx "vietpanel/panel/internal/app"
	"vietpanel/panel/internal/web"
)

var processManagerPage = web.LoadPage("system/process_manager.html")

// ProcessManagerPageData is system/process_manager.html's template context.
type ProcessManagerPageData struct {
	web.LayoutData
	ProcessData []Process
}

func renderProcessManagerPage(a *appctx.App, w http.ResponseWriter, r *http.Request, processes []Process) {
	layout, _, err := web.BuildLayoutData(a, w, r, "Process Manager")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	layout.BulkActions = processBulkActions(layout.T)
	data := ProcessManagerPageData{LayoutData: layout, ProcessData: processes}
	if err := processManagerPage.Render(w, http.StatusOK, data); err != nil {
		log.Printf("PROCESSMANAGER - template render error: %v", err)
	}
}

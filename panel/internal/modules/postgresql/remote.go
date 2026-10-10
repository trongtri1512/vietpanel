package postgresql

import (
	"net/http"
	"regexp"
	"strings"

	appctx "vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/core/logger"
	"vietpanel/panel/internal/core/reqip"
	"vietpanel/panel/internal/core/webserver"
	"vietpanel/panel/internal/modules/docker"
	"vietpanel/panel/internal/web"
)

var postgresPortRE = regexp.MustCompile(`^(127\.0\.0\.1:)?(\d+):5432$`)

// handleRemotePostgres toggles PostgreSQL's exposed port between bound to 127.0.0.1 (disabled) and 0.0.0.0 (enabled for remote access)
func handleRemotePostgres(a *appctx.App, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, currentUsername, userContext, err := auth.Injected(a, r)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	serverIP := a.GetCachedIPForUserOrPublicIPv4(ctx, currentUsername)

	postgresRemotePortOriginal := webserver.GetEnvFileValue(userContext, "POSTGRES_PORT")
	m := postgresPortRE.FindStringSubmatch(postgresRemotePortOriginal)
	if m == nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	remotePostgresPortOnly := m[2]

	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		ipAddress := reqip.ClientIP(r)
		portChanged := false
		switch r.Form.Get("action") {
		case "enable":
			docker.SetEnvValue(userContext, "POSTGRES_PORT", remotePostgresPortOnly+":5432")
			_ = logger.RecordUserAction(a.Config, currentUsername, "enabled remote PostgreSQL", ipAddress)
			web.Flash(a, w, r, "success", "Remote PostgreSQL access is now enabled.")
			portChanged = true
		case "disable":
			if strings.Contains(postgresRemotePortOriginal, "127.0.0.1") {
				web.Flash(a, w, r, "info", "Remote PostgreSQL access is already disabled.")
			} else {
				docker.SetEnvValue(userContext, "POSTGRES_PORT", "127.0.0.1:"+remotePostgresPortOnly+":5432")
				_ = logger.RecordUserAction(a.Config, currentUsername, "disabled remote PostgreSQL", ipAddress)
				web.Flash(a, w, r, "success", "Remote PostgreSQL access is now disabled.")
				portChanged = true
			}
		}
		if portChanged {
			if result := docker.RestartContainer(ctx, userContext, "postgres"); !result.Success {
				web.Flash(a, w, r, "error", "Port changed but the PostgreSQL service failed to restart. Try restarting it manually from Services.")
			}
		} else {
			docker.StartComposeServiceIfNotRunning(ctx, userContext, "postgres")
		}
	}

	postgresRemotePortOriginal = webserver.GetEnvFileValue(userContext, "POSTGRES_PORT")
	display := "ON"
	if strings.Contains(postgresRemotePortOriginal, "127.0.0.1") {
		display = "OFF"
	}

	if r.URL.Query().Get("output") == "json" {
		web.WriteJSON(w, http.StatusOK, map[string]any{
			"remote_postgresql_display": display, "server_ip": serverIP,
			"container_port": remotePostgresPortOnly, "postgres_port": 5432,
		})
		return
	}

	renderRemotePostgresPage(a, w, r, serverIP, remotePostgresPortOnly, display)
}

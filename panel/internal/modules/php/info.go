package php

import (
	"context"
	"net/http"
	"os/exec"
	"strings"
	"time"

	appctx "vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
	"vietpanel/panel/internal/core/podmanmanager"
	"vietpanel/panel/internal/core/webserver"
	"vietpanel/panel/internal/modules/docker"
	"vietpanel/panel/internal/web"
)

// handlePHPInfo renders `php -i` output for one PHP version, starting its container first if needed
func handlePHPInfo(a *appctx.App, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_, _, userContext, err := auth.Injected(a, r)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	version := phpVersionFromSegment(r.PathValue("phpversion"))
	title := "PHP INFO"
	var phpInfoContent string
	installedVersions := FetchPHPVersions(ctx, a, userContext)
	service := ""

	if version != "" {
		webServer := webserver.GetEnvFileValue(userContext, "WEB_SERVER")
		if strings.Contains(strings.ToLower(webServer), "litespeed") {
			service = strings.ToLower(webServer)
		} else {
			service = "php-fpm-" + version
		}
		title = "PHP " + version + " Info"

		if !docker.IsServiceRunning(ctx, userContext, service) {
			result := docker.StartOrStopContainer(ctx, userContext, service, "activate", "run")
			if !result.Success || !docker.IsServiceRunning(ctx, userContext, service) {
				web.FlashRedirect(a, w, r, "error", web.Tr(a, r, "Failed to start PHP %(service)s: %(message)s", "service", service, "message", result.Message), "/php/domains")
				return
			}
		}

		argv := podmanmanager.PodmanArgv(userContext, "exec", service, "php",
			"-d", "memory_limit=-1",
			"-d", "open_basedir=none",
			"-d", "disable_functions=",
			"-d", "display_errors=0",
			"-d", "error_log=/dev/null",
			"-i")
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		cmd := podmanmanager.Command(cctx, userContext, argv)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, runErr := cmd.Output()
		if runErr != nil {
			msg := stderr.String()
			if msg == "" {
				if ee, ok := runErr.(*exec.ExitError); ok {
					msg = string(ee.Stderr)
				} else {
					msg = runErr.Error()
				}
			}
			writeJSONError(w, http.StatusInternalServerError, strings.TrimSpace(msg))
			return
		}
		phpInfoContent = strings.TrimSpace(string(out))
	}

	renderPHPInfoPage(a, w, r, version, service, title, phpInfoContent, installedVersions)
}

// Package serverinfo implements the server information page, current/historical resource usage pages, and the /json/system/hosting/* endpoints those pages fetch from.
package serverinfo

import (
	"net/http"
	"os/exec"
	"regexp"
	"strings"

	appctx "vietpanel/panel/internal/app"
	"vietpanel/panel/internal/auth"
)

func injected(a *appctx.App, r *http.Request) (data map[string]any, err error) {
	userID, _ := auth.UserID(r)
	return a.InjectData(r.Context(), userID)
}

var (
	uptimeRE  = regexp.MustCompile(`up (.*?),`)
	loadAvgRE = regexp.MustCompile(`load average: (.*)$`)
)

// getUptimeAndLoad shells out to `uptime` and regex-extracts the two pieces callers actually want
func getUptimeAndLoad() (uptime, loadAvg string) {
	out, err := exec.Command("uptime").Output()
	if err != nil {
		return err.Error(), ""
	}
	output := strings.TrimSpace(string(out))
	if m := uptimeRE.FindStringSubmatch(output); m != nil {
		uptime = strings.TrimSpace(m[1])
	}
	if m := loadAvgRE.FindStringSubmatch(output); m != nil {
		loadAvg = strings.TrimSpace(m[1])
	}
	return uptime, loadAvg
}

// humanValue is the shared {pct, human} shape used throughout resource_usage.txt's JSON lines
type humanValue struct {
	Pct   float64 `json:"pct"`
	Human string  `json:"human"`
}

type humanOnly struct {
	Human string `json:"human"`
}

// ResourceUsageLine is the shared per-line schema of /home/<context>/resource_usage.txt, used by both the current-usage page (last line only) and the usage-history page (every line)
type ResourceUsageLine struct {
	Timestamp string `json:"timestamp"`
	CPU       struct {
		Usage humanValue `json:"usage"`
		Total humanValue `json:"total"`
	} `json:"cpu"`
	Memory struct {
		UsagePct  float64   `json:"usage_pct"`
		Used      humanOnly `json:"used"`
		Total     humanOnly `json:"total"`
		Available humanOnly `json:"available"`
	} `json:"memory"`
	Bandwidth struct {
		TotalSent humanOnly `json:"total_sent"`
		Limit     humanOnly `json:"limit"`
		UsagePct  float64   `json:"usage_pct"`
	} `json:"bandwidth"`
	Warning string `json:"warning"`
}

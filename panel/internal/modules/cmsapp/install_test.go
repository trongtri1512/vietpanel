package cmsapp

import (
	"net/http/httptest"
	"strings"
	"testing"

	appctx "vietpanel/panel/internal/app"
	"vietpanel/panel/internal/core/i18n"
	"vietpanel/panel/internal/web"
)

func TestRenderInstallPageEnforcedPrefix(t *testing.T) {
	mgr := i18n.NewManager(t.TempDir(), nil)
	app := New(App{Slug: "joomla", Name: "Joomla"})
	allowed := map[string]bool{"dashboard": true, "joomla": true}
	data := InstallPageData{
		LayoutData: web.LayoutData{
			Title: "Test", CSRFToken: "t", PanelDir: "ltr", UserAllowed: allowed, UserAllowedJSON: web.UserAllowedList(allowed),
			NavGroups: web.BuildClassicSidebarNav(allowed, nil, "/joomla/install"), RequestPath: "/joomla/install",
			PasswordStrength: 50, MySQLPrefix: "john_", T: mgr.Translator("en"),
		},
		Domains: []appctx.Domain{{DomainID: 1, Docroot: "/var/www/html/example.com", DomainURL: "example.com"}},
		App:     web.AppNames{Slug: "joomla", Name: "Joomla"},
	}
	w := httptest.NewRecorder()
	if err := app.installPage.Render(w, 200, data); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(w.Body.String(), ">john_</span>") {
		t.Error("expected the john_ prefix next to the db fields")
	}
}

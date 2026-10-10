package modules

import (
	"net/http"

	"vietpanel/panel/internal/app"
	"vietpanel/panel/internal/modules/account"
	"vietpanel/panel/internal/modules/autoinstaller"
	"vietpanel/panel/internal/modules/backups"
	"vietpanel/panel/internal/modules/cache"
	"vietpanel/panel/internal/modules/crons"
	"vietpanel/panel/internal/modules/dashboard"
	"vietpanel/panel/internal/modules/diskusage"
	"vietpanel/panel/internal/modules/dns"
	"vietpanel/panel/internal/modules/docker"
	"vietpanel/panel/internal/modules/domains"
	"vietpanel/panel/internal/modules/emails"
	"vietpanel/panel/internal/modules/filemanager"
	"vietpanel/panel/internal/modules/ftp"
	"vietpanel/panel/internal/modules/mysql"
	"vietpanel/panel/internal/modules/nodejs"
	"vietpanel/panel/internal/modules/php"
	"vietpanel/panel/internal/modules/python"
	"vietpanel/panel/internal/modules/security"
	"vietpanel/panel/internal/modules/services"
	"vietpanel/panel/internal/modules/websites"
)

func Register(mux *http.ServeMux, a *app.App) {
	account.Register(mux, a)
	dashboard.Register(mux, a)
	websites.Register(mux, a)
	domains.Register(mux, a)
	mysql.Register(mux, a)
	filemanager.Register(mux, a)
	backups.Register(mux, a)
	dns.Register(mux, a)
	emails.Register(mux, a)
	ftp.Register(mux, a)
	crons.Register(mux, a)
	services.Register(mux, a)
	docker.Register(mux, a)
	php.Register(mux, a)
	nodejs.Register(mux, a)
	python.Register(mux, a)
	cache.Register(mux, a)
	security.Register(mux, a)
	diskusage.Register(mux, a)
	autoinstaller.Register(mux, a)
}
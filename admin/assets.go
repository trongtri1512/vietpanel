// Package admin embeds the admin panel's templates and static assets into
// the binary so the whole panel ships as one file.
package admin

import "embed"

//go:embed static internal/webtemplates
var FS embed.FS

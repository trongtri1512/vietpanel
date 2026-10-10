// Package panel embeds the user panel's templates and static assets into the
// binary so the whole panel ships as one file.
package panel

import "embed"

//go:embed static internal/web/templates
var FS embed.FS

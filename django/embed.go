package django

import "embed"

//go:embed all:template
var embeddedFS embed.FS

package django

import "embed"

//go:embed template/**/*.django template/*.django
var embededFS embed.FS

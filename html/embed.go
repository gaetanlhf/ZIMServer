package html

import "embed"

//go:embed static/templates/*.html
var TemplatesFS embed.FS

//go:embed static/assets
var AssetsFS embed.FS

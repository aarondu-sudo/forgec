package template

import "embed"

// FS embeds all generator templates.
//go:embed *.tmpl
var FS embed.FS


// Package assets provides embedded static assets for sketch-talk.
package assets

import "embed"

//go:embed fonts/*.ttf
var FontFS embed.FS

//go:embed web
var WebFS embed.FS

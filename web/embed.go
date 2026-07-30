package web

import "embed"

//go:embed index.html templates static
var Files embed.FS

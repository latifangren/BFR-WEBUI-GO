package web

import "embed"

//go:embed index.html manifest.json sw.js templates static
var Files embed.FS

package web

import "embed"

//go:embed index.html manifest.json sw.js openapi.json templates static
var Files embed.FS

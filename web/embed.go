package web

import "embed"

//go:embed index.html static/*
var Files embed.FS

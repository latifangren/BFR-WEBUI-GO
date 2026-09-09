package frontend

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// GetFS returns an fs.FS rooted inside the dist directory
func GetFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

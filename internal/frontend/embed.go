package frontend

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var embeddedFS embed.FS

// GetFS returns the embedded frontend filesystem
func GetFS() (fs.FS, error) {
	return fs.Sub(embeddedFS, "static")
}
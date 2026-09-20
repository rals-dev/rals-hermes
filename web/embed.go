// Package web embeds the built Vue application (web/dist) into the binary.
//
// Run `npm run build` in this directory (or `make web`) before `go build`;
// without a build only the .gitkeep placeholder is embedded and the BFF
// answers "/" with a JSON 404 explaining that the UI is not bundled.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the built site rooted at index.html, and whether a build is
// actually present.
func FS() (fs.FS, bool) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return sub, false
	}
	return sub, true
}

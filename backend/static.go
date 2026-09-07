package main

import (
	"embed"
	"io/fs"
)

//go:embed all:static
var embeddedStatic embed.FS

// staticFS strips the "static" prefix so paths match what the browser
// requests (e.g. "index.html" instead of "static/index.html"). It's empty
// in local dev - see static/.gitkeep - which is fine since nothing routes
// to it there.
func staticFS() fs.FS {
	sub, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		panic("static.go: " + err.Error()) // the embed directive guarantees "static" exists
	}
	return sub
}

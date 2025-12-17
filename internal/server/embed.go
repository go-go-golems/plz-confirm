package server

import (
	"embed"
	"io/fs"
)

//go:embed embed/public
var embeddedFS embed.FS

// embeddedPublicFS is the filesystem for serving embedded static files.
// It strips the "embed/public" prefix so paths match what the frontend expects.
var embeddedPublicFS, _ = fs.Sub(embeddedFS, "embed/public")


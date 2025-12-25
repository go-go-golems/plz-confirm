//go:build !embed
// +build !embed

package server

import "io/fs"

// embeddedPublicFS is nil in default (dev/test) builds. In production builds we compile
// with `-tags embed` and generate `internal/server/embed/public` so assets can be embedded.
var embeddedPublicFS fs.FS = nil

//go:build !embed
// +build !embed

package server

import (
	"io/fs"
	"os"
	"path/filepath"
)

// embeddedPublicFS is nil unless we detect a local dev build output on disk.
//
// In default (no build tag) builds we try a best-effort disk fallback so
// `go run ./cmd/plz-confirm serve` can still serve the SPA after `go generate
// ./internal/server` has populated `internal/server/embed/public/`.
var embeddedPublicFS fs.FS = diskPublicFS()

func diskPublicFS() fs.FS {
	repoRoot, err := findRepoRootFromCWD()
	if err != nil {
		return nil
	}

	publicDir := filepath.Join(repoRoot, "internal", "server", "embed", "public")
	if _, err := os.Stat(filepath.Join(publicDir, "index.html")); err != nil {
		return nil
	}

	return os.DirFS(publicDir)
}

func findRepoRootFromCWD() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

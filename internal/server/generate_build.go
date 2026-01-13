//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"dagger.io/dagger"
)

func main() {
	// Find repo root (where go.mod is)
	repoRoot, err := findRepoRoot()
	if err != nil {
		fatalf("find repo root: %v", err)
	}

	// Paths
	frontendDir := filepath.Join(repoRoot, "agent-ui-system")
	embedDir := filepath.Join(repoRoot, "internal", "server", "embed", "public")

	if _, err := os.Stat(frontendDir); err != nil {
		fatalf("frontend directory not found at %s: %v", frontendDir, err)
	}

	pnpmVersion := os.Getenv("WEB_PNPM_VERSION")
	if pnpmVersion == "" {
		pnpmVersion = "10.4.1"
	}

	builderImage := os.Getenv("WEB_BUILDER_IMAGE")
	if builderImage == "" {
		builderImage = "node:22"
	}

	ctx := context.Background()
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		fatalf("connect dagger: %v", err)
	}
	defer client.Close()

	if err := os.RemoveAll(embedDir); err != nil {
		fatalf("remove %s: %v", embedDir, err)
	}
	if err := os.MkdirAll(embedDir, 0o755); err != nil {
		fatalf("mkdir %s: %v", embedDir, err)
	}

	uiDir := client.Host().Directory(frontendDir)
	ctr := client.Container().From(builderImage).
		WithWorkdir("/src").
		WithMountedDirectory("/src", uiDir).
		WithEnvVariable("PNPM_HOME", "/pnpm")

	analyticsEndpoint := os.Getenv("VITE_ANALYTICS_ENDPOINT")
	analyticsWebsiteID := os.Getenv("VITE_ANALYTICS_WEBSITE_ID")
	ctr = ctr.
		WithEnvVariable("VITE_ANALYTICS_ENDPOINT", analyticsEndpoint).
		WithEnvVariable("VITE_ANALYTICS_WEBSITE_ID", analyticsWebsiteID)

	if pnpmCacheDir := os.Getenv("PNPM_CACHE_DIR"); pnpmCacheDir != "" {
		if err := os.MkdirAll(pnpmCacheDir, 0o755); err != nil {
			fatalf("mkdir %s: %v", pnpmCacheDir, err)
		}
		cacheDir := client.Host().Directory(pnpmCacheDir)
		ctr = ctr.WithMountedDirectory("/pnpm/store", cacheDir).
			WithEnvVariable("PNPM_STORE_DIR", "/pnpm/store")
	}

	if os.Getenv("WEB_BUILDER_IMAGE") == "" || !strings.Contains(builderImage, ":") {
		ctr = ctr.WithExec([]string{
			"sh", "-lc",
			fmt.Sprintf("corepack enable && corepack prepare pnpm@%s --activate", pnpmVersion),
		})
	}

	ctr = ctr.
		WithExec([]string{"sh", "-lc", "pnpm --version"}).
		WithExec([]string{"sh", "-lc", "pnpm install --reporter=append-only"}).
		WithExec([]string{"sh", "-lc", "pnpm build"})

	dist := ctr.Directory("/src/dist/public")
	if _, err := dist.Export(ctx, embedDir); err != nil {
		fatalf("export dist: %v", err)
	}
	log.Printf("exported web dist to %s", embedDir)
}

func findRepoRoot() (string, error) {
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
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

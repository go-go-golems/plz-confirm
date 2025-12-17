// +build ignore

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	if err := buildAndCopy(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildAndCopy() error {
	// Find repo root (where go.mod is)
	repoRoot, err := findRepoRoot()
	if err != nil {
		return fmt.Errorf("find repo root: %w", err)
	}

	// Paths
	frontendDir := filepath.Join(repoRoot, "agent-ui-system")
	distDir := filepath.Join(frontendDir, "dist", "public")
	embedDir := filepath.Join(repoRoot, "internal", "server", "embed", "public")

	// Step 1: Build frontend with Vite
	fmt.Println("Building frontend with Vite...")
	buildCmd := exec.Command("pnpm", "run", "build")
	buildCmd.Dir = frontendDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("vite build failed: %w", err)
	}

	// Step 2: Ensure embed directory exists and is clean
	if err := os.RemoveAll(embedDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove embed dir: %w", err)
	}
	if err := os.MkdirAll(embedDir, 0755); err != nil {
		return fmt.Errorf("create embed dir: %w", err)
	}

	// Step 3: Copy dist/public to internal/server/embed/public
	fmt.Printf("Copying %s to %s...\n", distDir, embedDir)
	if err := copyDir(distDir, embedDir); err != nil {
		return fmt.Errorf("copy dist to embed: %w", err)
	}

	fmt.Println("Frontend build and copy completed successfully!")
	return nil
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

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}


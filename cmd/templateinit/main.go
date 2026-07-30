package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	app := flag.String("app", "", "application/binary name")
	module := flag.String("module", "", "Go module path")
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	if *app == "" || *module == "" {
		fatal("--app and --module are required")
	}
	goMod := filepath.Join(*root, "go.mod")
	b, err := os.ReadFile(goMod)
	if err != nil {
		fatal(err.Error())
	}
	oldModule := ""
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "module ") {
			oldModule = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			break
		}
	}
	if oldModule == "" {
		fatal("module line not found")
	}
	changes := 0
	err = filepath.WalkDir(*root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isTextFile(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		updated := strings.ReplaceAll(text, oldModule, *module)
		updated = strings.ReplaceAll(updated, `const appName = "app"`, `const appName = "`+*app+`"`)
		updated = strings.ReplaceAll(updated, "app_name: app", "app_name: "+*app)
		updated = strings.ReplaceAll(updated, "APP_BINARY_NAME: app", "APP_BINARY_NAME: "+*app)
		updated = strings.ReplaceAll(updated, "APP_BINARY_NAME:-app", "APP_BINARY_NAME:-"+*app)
		updated = strings.ReplaceAll(updated, "project_name: app", "project_name: "+*app)
		updated = strings.ReplaceAll(updated, "binary: app\n", "binary: "+*app+"\n")
		updated = strings.ReplaceAll(updated, "name_template: app_", "name_template: "+*app+"_")
		if updated == text {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(updated), info.Mode().Perm()); err != nil {
			return err
		}
		changes++
		return nil
	})
	if err != nil {
		fatal(err.Error())
	}
	fmt.Printf("Initialized app=%s module=%s (%d files changed)\n", *app, *module, changes)
}

func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return true
	}
	switch ext {
	case ".go", ".mod", ".sum", ".md", ".yml", ".yaml", ".json", ".sh", ".txt", ".example":
		return true
	default:
		return false
	}
}
func fatal(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }

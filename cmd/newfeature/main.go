package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var valid = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func main() {
	name := flag.String("name", "", "feature package name")
	flag.Parse()
	if !valid.MatchString(*name) {
		fatal("NAME must match ^[a-z][a-z0-9_]*$")
	}
	dir := filepath.Join("internal", "features", *name)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		fatal("feature already exists: " + dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatal(err.Error())
	}
	title := strings.ReplaceAll(*name, "_", " ")
	src := fmt.Sprintf(`package %s

import (
    "context"
    "fmt"

    "github.com/your-org/go-selfupdate-template/internal/features"
    "github.com/your-org/go-selfupdate-template/internal/tray"
)

type Feature struct{}
func (*Feature) ID() string { return %q }
func (*Feature) TrayItems() []tray.Item { return []tray.Item{{Title: %q, Action: %q, Enabled: true}} }
func (*Feature) Commands() []features.Command { return []features.Command{{Name: %q, Description: %q, Run: func(context.Context, []string) error { fmt.Println(%q); return nil }}} }
func (*Feature) Start(context.Context) error { return nil }
func init() { features.Register(&Feature{}) }
`, *name, *name, title, *name, *name, title+" feature", title+" feature is active")
	test := fmt.Sprintf(`package %s
import "testing"
func TestFeatureID(t *testing.T) { if (&Feature{}).ID() != %q { t.Fatal("unexpected feature ID") } }
`, *name, *name)
	mustWrite(filepath.Join(dir, *name+".go"), src)
	mustWrite(filepath.Join(dir, *name+"_test.go"), test)
	updateImports(*name)
	fmt.Println("created feature", *name)
}

func updateImports(name string) {
	path := filepath.Join("cmd", "app", "features_gen.go")
	b, _ := os.ReadFile(path)
	var imports []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `_ "`) {
			imports = append(imports, strings.Trim(strings.TrimPrefix(line, "_ "), `"`))
		}
	}
	imports = append(imports, "github.com/your-org/go-selfupdate-template/internal/features/"+name)
	sort.Strings(imports)
	var out strings.Builder
	out.WriteString("package main\n\nimport (\n")
	for _, p := range imports {
		fmt.Fprintf(&out, "\t_ %q\n", p)
	}
	out.WriteString(")\n")
	mustWrite(path, out.String())
}
func mustWrite(path, value string) {
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		fatal(err.Error())
	}
}
func fatal(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }

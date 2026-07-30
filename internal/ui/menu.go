package ui

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/your-org/go-selfupdate-template/internal/updater"
)

type row struct {
	version   string
	sources   []string
	changelog string
}

func Choose(in io.Reader, out io.Writer, releases []updater.Release, cached map[string]bool) (string, error) {
	if len(releases) == 0 {
		return "", fmt.Errorf("no releases available")
	}
	byVersion := map[string]*row{}
	var rows []*row
	for _, release := range releases {
		r := byVersion[release.Version]
		if r == nil {
			r = &row{version: release.Version, changelog: release.Changelog}
			byVersion[release.Version] = r
			rows = append(rows, r)
		}
		if !contains(r.sources, release.Source) {
			r.sources = append(r.sources, release.Source)
		}
	}
	fmt.Fprintln(out, "Available versions:")
	for i, r := range rows {
		sort.Strings(r.sources)
		cacheMark := "-"
		if cached[r.version] {
			cacheMark = "✓"
		}
		fmt.Fprintf(out, "%d) %s sources=%s cache=%s %s\n", i+1, r.version, strings.Join(r.sources, ","), cacheMark, firstLine(r.changelog))
	}
	fmt.Fprint(out, "Select version (q to cancel): ")
	s, _ := bufio.NewReader(in).ReadString('\n')
	s = strings.TrimSpace(s)
	if s == "q" || s == "" {
		return "", nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > len(rows) {
		return "", fmt.Errorf("invalid selection")
	}
	return rows[n-1].version, nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 80 {
		s = s[:80] + "..."
	}
	return s
}

package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/go-selfupdate-template/internal/updater"
)

func TestChooseGroupsSourcesAndShowsCache(t *testing.T) {
	releases := []updater.Release{
		{Version: "1.26.0729.2000", Source: "github", Changelog: "fixed updater"},
		{Version: "1.26.0729.2000", Source: "azure"},
	}
	var out bytes.Buffer
	got, err := Choose(strings.NewReader("1\n"), &out, releases, map[string]bool{"1.26.0729.2000": true})
	if err != nil || got != "1.26.0729.2000" {
		t.Fatalf("got=%s err=%v", got, err)
	}
	text := out.String()
	if !strings.Contains(text, "sources=azure,github") || !strings.Contains(text, "cache=✓") {
		t.Fatalf("output=%q", text)
	}
}

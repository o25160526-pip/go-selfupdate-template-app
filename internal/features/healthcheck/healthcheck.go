package healthcheck

import (
	"context"
	"fmt"

	"github.com/your-org/go-selfupdate-template/internal/features"
	"github.com/your-org/go-selfupdate-template/internal/tray"
)

type Feature struct{}

func (*Feature) ID() string { return "healthcheck" }
func (*Feature) TrayItems() []tray.Item {
	return []tray.Item{{Title: "healthcheck", Action: "healthcheck", Enabled: true}}
}
func (*Feature) Commands() []features.Command {
	return []features.Command{{Name: "healthcheck", Description: "healthcheck feature", Run: func(context.Context, []string) error { fmt.Println("healthcheck feature is active"); return nil }}}
}
func (*Feature) Start(context.Context) error { return nil }
func init()                                  { features.Register(&Feature{}) }

package sample

import (
	"context"
	"fmt"
	"github.com/your-org/go-selfupdate-template/internal/features"
	"github.com/your-org/go-selfupdate-template/internal/tray"
)

type Feature struct{}

func (*Feature) ID() string { return "sample" }
func (*Feature) TrayItems() []tray.Item {
	return []tray.Item{{Title: "Sample status", Action: "sample status"}}
}
func (*Feature) Commands() []features.Command {
	return []features.Command{{Name: "sample", Description: "example feature command", Run: func(context.Context, []string) error { fmt.Println("sample feature is active"); return nil }}}
}
func (*Feature) Start(context.Context) error { return nil }
func init()                                  { features.Register(&Feature{}) }

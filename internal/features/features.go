package features

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/your-org/go-selfupdate-template/internal/tray"
)

type Command struct {
	Name, Description string
	Run               func(context.Context, []string) error
}
type Feature interface {
	ID() string
	TrayItems() []tray.Item
	Commands() []Command
	Start(context.Context) error
}

var (
	mu       sync.RWMutex
	registry = map[string]Feature{}
)

func Register(f Feature) {
	mu.Lock()
	defer mu.Unlock()
	if f == nil || f.ID() == "" {
		panic("feature ID required")
	}
	if _, ok := registry[f.ID()]; ok {
		panic(fmt.Sprintf("duplicate feature %s", f.ID()))
	}
	registry[f.ID()] = f
}
func All() []Feature {
	mu.RLock()
	defer mu.RUnlock()
	ids := make([]string, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Feature, 0, len(ids))
	for _, id := range ids {
		out = append(out, registry[id])
	}
	return out
}

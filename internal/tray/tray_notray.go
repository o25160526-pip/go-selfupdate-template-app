//go:build !tray

package tray

import "fmt"

func Run([]Item) error {
	return fmt.Errorf("tray support is not included; rebuild with -tags tray on a desktop host")
}

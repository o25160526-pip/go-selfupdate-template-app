//go:build tray

package tray

import (
	"bufio"
	"fmt"
	"os"
)

// Run is a dependency-free desktop-safe fallback used by the template build.
// Replace this adapter with fyne.io/systray when GUI dependencies are available.
func Run(items []Item) error {
	fmt.Println("Tray adapter menu:")
	for i, x := range items {
		fmt.Printf("%d. %s\n", i+1, x.Title)
	}
	fmt.Print("Press Enter to exit tray adapter...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	return nil
}

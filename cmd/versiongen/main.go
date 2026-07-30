package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/your-org/go-selfupdate-template/internal/version"
)

func main() {
	format := flag.String("format", "display", "display|tag|both")
	at := flag.String("time", "", "RFC3339 UTC override")
	flag.Parse()
	now := time.Now().UTC()
	if *at != "" {
		var err error
		now, err = time.Parse(time.RFC3339, *at)
		if err != nil {
			fatal(err)
		}
		now = now.UTC()
	}
	v := version.Version{Major: 1, Year: now.Year() % 100, Month: int(now.Month()), Day: now.Day(), Hour: now.Hour(), Minute: now.Minute()}
	for tagExists(v.Tag()) {
		now = now.Add(time.Minute)
		v = version.Version{Major: 1, Year: now.Year() % 100, Month: int(now.Month()), Day: now.Day(), Hour: now.Hour(), Minute: now.Minute()}
	}
	switch *format {
	case "display":
		fmt.Println(v.String())
	case "tag":
		fmt.Println(v.Tag())
	case "both":
		fmt.Printf("DISPLAY=%s\nTAG=%s\n", v.String(), v.Tag())
	default:
		fatal(fmt.Errorf("unknown format %s", *format))
	}
}
func tagExists(tag string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "refs/tags/"+tag)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}
func fatal(err error) { fmt.Fprintln(os.Stderr, strings.TrimSpace(err.Error())); os.Exit(1) }

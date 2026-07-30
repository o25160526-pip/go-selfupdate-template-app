package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/your-org/go-selfupdate-template/internal/config"
	"github.com/your-org/go-selfupdate-template/internal/features"
	"github.com/your-org/go-selfupdate-template/internal/tray"
	"github.com/your-org/go-selfupdate-template/internal/ui"
	"github.com/your-org/go-selfupdate-template/internal/updater"
	appversion "github.com/your-org/go-selfupdate-template/internal/version"
)

const appName = "app"

func Run(args []string, in io.Reader, out, errOut io.Writer) int {
	configPath, args := extractConfig(args)
	cfg, err := config.Load(appName, configPath)
	if err != nil {
		fmt.Fprintln(errOut, "config:", err)
		return 2
	}
	if len(args) == 0 {
		usage(out)
		return 0
	}
	for _, f := range features.All() {
		if err := f.Start(context.Background()); err != nil {
			fmt.Fprintf(errOut, "feature %s: %v\n", f.ID(), err)
			return 1
		}
	}
	switch args[0] {
	case "version":
		return runVersion(args[1:], out, errOut)
	case "update":
		return runUpdate(cfg, args[1:], in, out, errOut)
	case "rollback":
		return runRollback(cfg, args[1:], out, errOut)
	case "channel":
		return runChannel(cfg, configPath, args[1:], out, errOut)
	case "cache":
		return runCache(cfg, args[1:], out, errOut)
	case "config":
		if len(args) > 1 && args[1] == "show" {
			fmt.Fprintln(out, cfg.JSON())
			return 0
		}
		fmt.Fprintln(errOut, "usage: app config show")
		return 2
	case "menu":
		return runMenu(cfg, in, out, errOut)
	case "tray":
		return runTray(out, errOut)
	case "features":
		for _, f := range features.All() {
			fmt.Fprintln(out, f.ID())
		}
		return 0
	default:
		for _, f := range features.All() {
			for _, c := range f.Commands() {
				if c.Name == args[0] {
					if err := c.Run(context.Background(), args[1:]); err != nil {
						fmt.Fprintln(errOut, err)
						return 1
					}
					return 0
				}
			}
		}
		fmt.Fprintf(errOut, "unknown command %q\n", args[0])
		usage(errOut)
		return 2
	}
}
func extractConfig(args []string) (string, []string) {
	path := os.Getenv("APP_CONFIG")
	var out []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--config=") {
			path = strings.TrimPrefix(args[i], "--config=")
			continue
		}
		if args[i] == "--config" && i+1 < len(args) {
			path = args[i+1]
			i++
			continue
		}
		out = append(out, args[i])
	}
	return path, out
}
func usage(w io.Writer) {
	fmt.Fprintln(w, `Usage: app [--config path] <command>
  version [--json]
  update [--version X|--latest] [--silent] [--list] [--dry-run] [--channel stable|beta|internal] [--timeout 5m]
  rollback
  channel set stable|beta|internal
  cache list | prune --keep 3 | prefetch --keep 3
  config show
  menu | tray | features
  sample`)
}
func runVersion(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(errOut)
	j := fs.Bool("json", false, "JSON output")
	if fs.Parse(args) != nil {
		return 2
	}
	if *j {
		b, _ := appversion.JSON()
		fmt.Fprintln(out, string(b))
	} else {
		fmt.Fprintln(out, appversion.Current().String())
	}
	return 0
}
func runUpdate(cfg config.Config, args []string, in io.Reader, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.SetOutput(errOut)
	ver := fs.String("version", "", "target version")
	latest := fs.Bool("latest", false, "latest")
	silent := fs.Bool("silent", false, "no prompt")
	list := fs.Bool("list", false, "list releases")
	dry := fs.Bool("dry-run", false, "resolve only")
	channel := fs.String("channel", cfg.Channel, "channel")
	timeout := fs.Duration("timeout", cfg.Timeout, "timeout")
	if fs.Parse(args) != nil {
		return 2
	}
	if *latest && *ver != "" {
		fmt.Fprintln(errOut, "--latest and --version are mutually exclusive")
		return 2
	}
	cfg.Channel = *channel
	if err := cfg.ValidateForUpdate(); err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	closer, log := openLog(cfg.LogFile)
	if closer != nil {
		defer closer.Close()
	}
	eng := makeEngine(cfg, log)
	ctx := context.Background()
	if *list {
		rels, fail, listErr := eng.ListWithPolicy(ctx)
		if listErr != nil {
			fmt.Fprintln(errOut, listErr)
			return updater.ExitVerify
		}
		for _, r := range rels {
			a, ae := updater.SelectAsset(r, cfg.AppName, runtime.GOOS, runtime.GOARCH)
			mark := ""
			if ae != nil {
				mark = " [no matching asset]"
			} else {
				mark = " " + a.Name
			}
			fmt.Fprintf(out, "%s\t%s\t%s%s\n", r.Version, r.Source, displayFlags(r), mark)
		}
		for _, f := range fail {
			fmt.Fprintln(errOut, "warning:", f.Error())
		}
		if len(rels) == 0 {
			return updater.ExitNotFound
		}
		return 0
	}
	var confirm func(updater.UpdateResult) bool
	// CI runners do not have a terminal. Never block waiting for input there;
	// retain the confirmation prompt for real interactive local use.
	if !*silent && !*dry && isInteractiveInput(in) {
		confirm = func(plan updater.UpdateResult) bool {
			fmt.Fprintf(out, "Update %s from %s to %s via %s? [y/N]: ", cfg.Channel, plan.From, plan.To, plan.Source)
			var answer string
			fmt.Fscanln(in, &answer)
			answer = strings.ToLower(strings.TrimSpace(answer))
			return answer == "y" || answer == "yes"
		}
	}
	result, err := eng.Update(ctx, updater.UpdateOptions{Version: *ver, DryRun: *dry, Timeout: *timeout, Confirm: confirm})
	if err != nil {
		if errors.Is(err, updater.ErrCancelled) {
			fmt.Fprintln(out, "cancelled")
			return 0
		}
		var ee *updater.ExitError
		if errors.As(err, &ee) {
			if ee.Code == updater.ExitUpToDate {
				fmt.Fprintln(out, ee.Err)
				return ee.Code
			}
			fmt.Fprintln(errOut, ee.Err)
			return ee.Code
		}
		fmt.Fprintln(errOut, err)
		return 1
	}
	if *dry {
		fmt.Fprintln(out, updater.ResultJSON(result))
	} else {
		fmt.Fprintf(out, "updated %s -> %s via %s (%s)\n", result.From, result.To, result.Source, result.Asset)
	}
	return 0
}

func isInteractiveInput(in io.Reader) bool {
	if in != os.Stdin {
		return false
	}
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func displayFlags(r updater.Release) string {
	var f []string
	if r.Draft {
		f = append(f, "draft")
	}
	if r.Prerelease {
		f = append(f, "prerelease")
	}
	if len(f) == 0 {
		return "stable"
	}
	return strings.Join(f, ",")
}
func makeEngine(cfg config.Config, log io.Writer) updater.Engine {
	var sources []updater.Source
	metadata := updater.MetadataCache{Root: cfg.CacheDir, TTL: 15 * time.Minute}
	if cfg.GitHubOwner != "" && cfg.GitHubRepo != "" {
		sources = append(sources, &updater.GitHubSource{Owner: cfg.GitHubOwner, Repo: cfg.GitHubRepo, APIBase: cfg.GitHubAPI, Client: http.DefaultClient, Metadata: metadata})
	}
	if cfg.AzureIndexURL != "" {
		sources = append(sources, &updater.AzureBlobSource{IndexURL: cfg.AzureIndexURL, Client: http.DefaultClient, Metadata: metadata})
	}
	exe, _ := os.Executable()
	return updater.Engine{AppName: cfg.AppName, CurrentVersion: appversion.Current().String(), TargetPath: exe, Channel: cfg.Channel, Token: cfg.UpdateToken, ManifestURL: cfg.ManifestURL, PublicKeys: cfg.PublicKeys, Sources: sources, Cache: updater.Cache{Root: cfg.CacheDir}, Metadata: metadata, Client: http.DefaultClient, Log: log}
}
func openLog(path string) (io.WriteCloser, io.Writer) {
	if path == "" {
		return nil, io.Discard
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, io.Discard
	}
	return f, f
}
func runRollback(cfg config.Config, args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("rollback", flag.ContinueOnError)
	fs.SetOutput(errOut)
	target := fs.String("target", "", "target executable for testing")
	if fs.Parse(args) != nil {
		return 2
	}
	if *target == "" {
		*target, _ = os.Executable()
	}
	if err := updater.WithLock(filepath.Join(cfg.CacheDir, "update.lock"), func() error { return updater.Rollback(*target, *target+".rollback") }); err != nil {
		fmt.Fprintln(errOut, err)
		return updater.ExitApplyRollback
	}
	fmt.Fprintln(out, "rollback completed")
	return 0
}
func runChannel(cfg config.Config, path string, args []string, out, errOut io.Writer) int {
	if len(args) != 2 || args[0] != "set" {
		fmt.Fprintln(errOut, "usage: app channel set stable|beta|internal")
		return 2
	}
	ch := args[1]
	if ch != "stable" && ch != "beta" && ch != "internal" {
		fmt.Fprintln(errOut, "invalid channel")
		return 2
	}
	if path == "" {
		path = config.DefaultPath(appName)
	}
	if err := setConfigKey(path, "channel", ch); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintln(out, "channel set to", ch)
	return 0
}
func setConfigKey(path, key, value string) error {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	b, _ := os.ReadFile(path)
	lines := strings.Split(string(b), "\n")
	found := false
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), key+":") {
			lines[i] = key + ": " + value
			found = true
		}
	}
	if !found {
		lines = append(lines, key+": "+value)
	}
	return os.WriteFile(path, []byte(strings.TrimLeft(strings.Join(lines, "\n"), "\n")), 0o600)
}
func runCache(cfg config.Config, args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errOut, "usage: app cache list|prune|prefetch")
		return 2
	}
	cache := updater.Cache{Root: cfg.CacheDir}
	switch args[0] {
	case "list":
		v, err := cache.Load()
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		b, _ := json.MarshalIndent(v, "", "  ")
		fmt.Fprintln(out, string(b))
		return 0
	case "prune":
		fs := flag.NewFlagSet("prune", flag.ContinueOnError)
		fs.SetOutput(errOut)
		keep := fs.Int("keep", 3, "entries")
		if fs.Parse(args[1:]) != nil {
			return 2
		}
		v, err := cache.Prune(*keep)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		fmt.Fprintf(out, "removed %d entries\n", len(v))
		return 0
	case "prefetch":
		return runPrefetch(cfg, args[1:], out, errOut)
	default:
		fmt.Fprintln(errOut, "unknown cache command")
		return 2
	}
}
func runPrefetch(cfg config.Config, args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("prefetch", flag.ContinueOnError)
	fs.SetOutput(errOut)
	keep := fs.Int("keep", 3, "versions")
	if fs.Parse(args) != nil {
		return 2
	}
	if err := cfg.ValidateForUpdate(); err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	eng := makeEngine(cfg, io.Discard)
	rels, fail, listErr := eng.ListWithPolicy(context.Background())
	if listErr != nil {
		fmt.Fprintln(errOut, listErr)
		return updater.ExitVerify
	}
	for _, f := range fail {
		fmt.Fprintln(errOut, "warning:", f.Error())
	}
	count := 0
	cachedVersions := map[string]bool{}
	for _, r := range rels {
		if cachedVersions[r.Version] {
			continue
		}
		if count >= *keep {
			break
		}
		cmp, err := appversion.Compare(r.Version, appversion.Current().String())
		if err != nil || cmp <= 0 {
			continue
		}
		a, err := updater.SelectAsset(r, cfg.AppName, runtime.GOOS, runtime.GOARCH)
		if err != nil {
			continue
		}
		a, err = eng.PrepareAsset(context.Background(), r, a)
		if err != nil {
			fmt.Fprintln(errOut, err)
			continue
		}
		dst := filepath.Join(cfg.CacheDir, "staging", r.Source, a.Name)
		sha, err := (updater.Downloader{Retries: 3, Token: cfg.UpdateToken}).Download(context.Background(), a, dst, cfg.PublicKeys)
		if err != nil {
			fmt.Fprintln(errOut, err)
			continue
		}
		_, err = eng.Cache.Import(dst, sha, r.Version, a.Name)
		if err != nil {
			fmt.Fprintln(errOut, err)
			continue
		}
		fmt.Fprintln(out, "prefetched", r.Version, a.Name)
		cachedVersions[r.Version] = true
		count++
	}
	_, _ = eng.Cache.Prune(*keep)
	if count == 0 {
		return updater.ExitNotFound
	}
	return 0
}
func runMenu(cfg config.Config, in io.Reader, out, errOut io.Writer) int {
	if err := cfg.ValidateForUpdate(); err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	eng := makeEngine(cfg, io.Discard)
	rels, fail, listErr := eng.ListWithPolicy(context.Background())
	if listErr != nil {
		fmt.Fprintln(errOut, listErr)
		return updater.ExitVerify
	}
	for _, f := range fail {
		fmt.Fprintln(errOut, "warning:", f.Error())
	}
	cached := map[string]bool{}
	if entries, err := eng.Cache.Load(); err == nil {
		for _, entry := range entries {
			cached[entry.Version] = true
		}
	}
	v, err := ui.Choose(in, out, rels, cached)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if v == "" {
		return 0
	}
	return runUpdate(cfg, []string{"--version", v}, in, out, errOut)
}

func runTray(out, errOut io.Writer) int {
	items := []tray.Item{{Title: "Check now", Action: "update --dry-run", Enabled: true}, {Title: "Update now", Action: "update", Enabled: true}, {Title: "Open menu", Action: "menu", Enabled: true}, {Title: "Postpone 24h", Action: "postpone", Enabled: true}, {Title: "Quit", Action: "quit", Enabled: true}}
	for _, f := range features.All() {
		items = append(items, f.TrayItems()...)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Title < items[j].Title })
	if err := tray.Run(items); err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	fmt.Fprintln(out, "tray exited")
	return 0
}

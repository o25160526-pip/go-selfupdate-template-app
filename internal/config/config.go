package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/your-org/go-selfupdate-template/internal/buildinfo"
)

type Config struct {
	AppName       string        `json:"app_name"`
	Channel       string        `json:"channel"`
	GitHubOwner   string        `json:"github_owner"`
	GitHubRepo    string        `json:"github_repo"`
	GitHubAPI     string        `json:"github_api"`
	AzureIndexURL string        `json:"azure_index_url"`
	ManifestURL   string        `json:"manifest_url"`
	CacheDir      string        `json:"cache_dir"`
	LogFile       string        `json:"log_file"`
	Timeout       time.Duration `json:"timeout"`
	AutoUpdate    bool          `json:"auto_update"`
	PublicKeys    []string      `json:"public_keys"`
	UpdateToken   string        `json:"-"`
}

func Defaults(app string) Config {
	home, _ := os.UserHomeDir()
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		cacheRoot = filepath.Join(home, ".cache")
	}
	configRoot, err := os.UserConfigDir()
	if err != nil {
		configRoot = filepath.Join(home, ".config")
	}
	return Config{AppName: app, Channel: "stable", GitHubOwner: buildinfo.DefaultGitHubOwner, GitHubRepo: buildinfo.DefaultGitHubRepo, GitHubAPI: "https://api.github.com", AzureIndexURL: buildinfo.DefaultAzureIndexURL, ManifestURL: buildinfo.DefaultManifestURL, CacheDir: filepath.Join(cacheRoot, app), LogFile: filepath.Join(configRoot, app, "update.log"), Timeout: 5 * time.Minute, PublicKeys: buildinfo.PublicKeys()}
}

func DefaultPath(app string) string {
	root, err := os.UserConfigDir()
	if err != nil {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".config")
	}
	return filepath.Join(root, app, "config.yaml")
}

// Load applies: defaults < config file < APP_* environment. CLI overrides are applied by the caller.
func Load(app, path string) (Config, error) {
	c := Defaults(app)
	if path == "" {
		path = DefaultPath(app)
	}
	if b, err := os.ReadFile(path); err == nil {
		if err := parseYAMLSubset(b, &c); err != nil {
			return c, fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return c, err
	}
	applyEnv(&c)
	if err := c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}

func (c Config) Validate() error {
	switch c.Channel {
	case "stable", "beta", "internal":
	default:
		return fmt.Errorf("invalid channel %q", c.Channel)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

func (c Config) ValidateForUpdate() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Channel == "internal" && c.UpdateToken == "" {
		return fmt.Errorf("channel internal requires APP_UPDATE_TOKEN")
	}
	return nil
}

func applyEnv(c *Config) {
	set := func(name string, dst *string) {
		if v, ok := os.LookupEnv(name); ok {
			*dst = v
		}
	}
	set("APP_CHANNEL", &c.Channel)
	set("APP_GITHUB_OWNER", &c.GitHubOwner)
	set("APP_GITHUB_REPO", &c.GitHubRepo)
	set("APP_GITHUB_API", &c.GitHubAPI)
	set("APP_AZURE_INDEX_URL", &c.AzureIndexURL)
	set("APP_MANIFEST_URL", &c.ManifestURL)
	set("APP_CACHE_DIR", &c.CacheDir)
	set("APP_LOG_FILE", &c.LogFile)
	set("APP_UPDATE_TOKEN", &c.UpdateToken)
	if v := os.Getenv("APP_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		}
	}
	if v := os.Getenv("APP_AUTO_UPDATE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.AutoUpdate = b
		}
	}
	if v := os.Getenv("APP_PUBLIC_KEYS"); v != "" {
		c.PublicKeys = splitCSV(v)
	}
}

func parseYAMLSubset(b []byte, c *Config) error {
	s := bufio.NewScanner(strings.NewReader(string(b)))
	line := 0
	for s.Scan() {
		line++
		raw := strings.TrimSpace(s.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		p := strings.SplitN(raw, ":", 2)
		if len(p) != 2 {
			return fmt.Errorf("line %d: expected key: value", line)
		}
		key := strings.TrimSpace(p[0])
		val := strings.Trim(strings.TrimSpace(p[1]), "\"'")
		switch key {
		case "app_name":
			c.AppName = val
		case "channel":
			c.Channel = val
		case "github_owner":
			c.GitHubOwner = val
		case "github_repo":
			c.GitHubRepo = val
		case "github_api":
			c.GitHubAPI = val
		case "azure_index_url":
			c.AzureIndexURL = val
		case "manifest_url":
			c.ManifestURL = val
		case "cache_dir":
			c.CacheDir = expandHome(val)
		case "log_file":
			c.LogFile = expandHome(val)
		case "timeout":
			d, err := time.ParseDuration(val)
			if err != nil {
				return err
			}
			c.Timeout = d
		case "auto_update":
			v, err := strconv.ParseBool(val)
			if err != nil {
				return err
			}
			c.AutoUpdate = v
		case "public_keys":
			c.PublicKeys = splitCSV(strings.Trim(val, "[]"))
		default:
			return fmt.Errorf("line %d: unknown key %q", line, key)
		}
	}
	return s.Err()
}
func splitCSV(v string) []string {
	var out []string
	for _, x := range strings.Split(v, ",") {
		x = strings.Trim(strings.TrimSpace(x), "\"'")
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}
func expandHome(v string) string {
	if strings.HasPrefix(v, "~/") {
		h, _ := os.UserHomeDir()
		return filepath.Join(h, v[2:])
	}
	return v
}
func (c Config) JSON() string { b, _ := json.MarshalIndent(c, "", "  "); return string(b) }

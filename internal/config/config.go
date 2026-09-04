package config

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gofrs/flock"
	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultProfile = "default"
	DefaultConfig  = "config.toml"
	updateLockWait = 25 * time.Millisecond
)

type Config struct {
	DefaultProfile string             `toml:"default_profile"`
	Profiles       map[string]Profile `toml:"profile"`
}

type Profile struct {
	Browser            string `toml:"browser"`
	BrowserProfile     string `toml:"browser_profile"`
	CookiePath         string `toml:"cookie_path"`
	Auth               string `toml:"auth"`
	SpotifyClientID    string `toml:"spotify_client_id"`
	SpotifyRedirectURI string `toml:"spotify_redirect_uri"`
	Market             string `toml:"market"`
	Language           string `toml:"language"`
	Device             string `toml:"device"`
	Engine             string `toml:"engine"`
}

func DefaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "spogo", DefaultConfig), nil
}

func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return nil, err
	}
	cfg := Default()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.normalize()
	return cfg, nil
}

func Save(path string, cfg *Config) error {
	if cfg == nil {
		return errors.New("nil config")
	}
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	cfg.normalize()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Update serializes a load-modify-save transaction for the shared config file.
func Update(ctx context.Context, path string, fn func(*Config) error) (*Config, error) {
	if fn == nil {
		return nil, errors.New("nil config update")
	}
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	configLock := flock.New(path+".lock", flock.SetPermissions(0o600))
	locked, err := configLock.TryLockContext(ctx, updateLockWait)
	if err != nil {
		_ = configLock.Close()
		return nil, fmt.Errorf("lock config: %w", err)
	}
	if !locked {
		_ = configLock.Close()
		return nil, fmt.Errorf("lock config: %w", ctx.Err())
	}
	defer func() {
		_ = configLock.Unlock()
		_ = configLock.Close()
	}()
	if runtime.GOOS != "windows" {
		if err := os.Chmod(configLock.Path(), 0o600); err != nil {
			return nil, err
		}
	}

	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}
	if err := fn(cfg); err != nil {
		return nil, err
	}
	if err := Save(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Default() *Config {
	return &Config{
		DefaultProfile: DefaultProfile,
		Profiles:       map[string]Profile{},
	}
}

func (c *Config) Profile(name string) Profile {
	if c == nil {
		return Profile{}
	}
	if name == "" {
		name = c.DefaultProfile
	}
	if name == "" {
		name = DefaultProfile
	}
	if c.Profiles == nil {
		return Profile{}
	}
	return c.Profiles[name]
}

func (c *Config) SetProfile(name string, profile Profile) {
	if c == nil {
		return
	}
	if name == "" {
		name = DefaultProfile
	}
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	c.Profiles[name] = profile
}

func CookiePath(configPath, profile string) string {
	if profile == "" {
		profile = DefaultProfile
	}
	if configPath == "" {
		return ""
	}
	base := filepath.Dir(configPath)
	return filepath.Join(base, "cookies", profile+".json")
}

func CachePath(configPath, profile string) string {
	if profile == "" {
		profile = DefaultProfile
	}
	if configPath == "" {
		return ""
	}
	base := filepath.Dir(configPath)
	return filepath.Join(base, "cache", profile+".json")
}

func OAuthTokenPath(configPath, profile string) string {
	if profile == "" {
		profile = DefaultProfile
	}
	if configPath == "" {
		return ""
	}
	base := filepath.Dir(configPath)
	return filepath.Join(base, "oauth", oauthProfileFilename(profile))
}

func oauthProfileFilename(profile string) string {
	if isPortableProfileFilename(profile) {
		return profile + ".json"
	}
	return "~" + hex.EncodeToString([]byte(profile)) + ".json"
}

func isPortableProfileFilename(profile string) bool {
	if profile == "" || profile == "." || profile == ".." || strings.HasSuffix(profile, ".") {
		return false
	}
	for _, char := range profile {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.' {
			continue
		}
		return false
	}
	stem := strings.ToUpper(strings.SplitN(profile, ".", 2)[0])
	if stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" {
		return false
	}
	if len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) &&
		stem[3] >= '1' && stem[3] <= '9' {
		return false
	}
	return true
}

func (c *Config) normalize() {
	if c.DefaultProfile == "" {
		c.DefaultProfile = DefaultProfile
	}
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
}

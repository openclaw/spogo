package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultProfile = "default"
	DefaultConfig  = "config.toml"
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
	return filepath.Join(base, "oauth", profile+".json")
}

func (c *Config) normalize() {
	if c.DefaultProfile == "" {
		c.DefaultProfile = DefaultProfile
	}
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
}

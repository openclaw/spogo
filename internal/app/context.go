package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/steipete/spogo/internal/config"
	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
)

type Settings struct {
	ConfigPath         string
	Profile            string
	Timeout            time.Duration
	Market             string
	Language           string
	Device             string
	Engine             string
	Auth               string
	SpotifyClientID    string
	SpotifyRedirectURI string
	Format             output.Format
	NoColor            bool
	Quiet              bool
	Verbose            bool
	Debug              bool
	NoInput            bool
}

type Context struct {
	Settings   Settings
	Config     *config.Config
	ConfigPath string
	Profile    config.Profile
	ProfileKey string
	Output     *output.Writer

	spotifyClient spotify.API
	commandCtx    context.Context
}

func (c *Context) SetSpotify(client spotify.API) {
	if c == nil {
		return
	}
	c.spotifyClient = client
}

func (c *Context) SaveProfile(profile config.Profile) error {
	if c == nil {
		return errors.New("nil context")
	}
	if c.Config == nil {
		return errors.New("nil config")
	}
	cfg := c.Config
	cfg.SetProfile(c.ProfileKey, profile)
	cfg.DefaultProfile = c.ProfileKey
	if err := config.Save(c.ConfigPath, cfg); err != nil {
		return err
	}
	c.Profile = profile
	return nil
}

func (c *Context) ResolveCookiePath() string {
	return config.CookiePath(c.ConfigPath, c.ProfileKey)
}

func (c *Context) ResolveCachePath() string {
	return config.CachePath(c.ConfigPath, c.ProfileKey)
}

func (c *Context) ResolveOAuthTokenPath() string {
	return config.OAuthTokenPath(c.ConfigPath, c.ProfileKey)
}

func (c *Context) ClearCache() error {
	path := c.ResolveCachePath()
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (c *Context) EnsureTimeout() time.Duration {
	if c.Settings.Timeout > 0 {
		return c.Settings.Timeout
	}
	return 10 * time.Second
}

func (c *Context) CommandContext() context.Context {
	if c == nil || c.commandCtx == nil {
		return context.Background()
	}
	return c.commandCtx
}

func (c *Context) SetCommandContext(ctx context.Context) {
	if c == nil {
		return
	}
	if ctx == nil {
		c.commandCtx = context.Background()
		return
	}
	c.commandCtx = ctx
}

func (c *Context) ValidateProfile() error {
	if c.Profile.Market != "" && len(c.Profile.Market) != 2 {
		return fmt.Errorf("market must be 2-letter country code")
	}
	return nil
}

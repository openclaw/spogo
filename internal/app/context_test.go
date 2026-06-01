package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/steipete/spogo/internal/config"
	"github.com/steipete/spogo/internal/cookies"
	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
)

func TestNewContextLoadsProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := config.Default()
	cfg.SetProfile("work", config.Profile{Market: "US", Language: "en"})
	cfg.DefaultProfile = "work"
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	ctx, err := NewContext(Settings{ConfigPath: path, Format: output.FormatPlain})
	if err != nil {
		t.Fatalf("new context: %v", err)
	}
	if ctx.Profile.Market != "US" {
		t.Fatalf("market: %s", ctx.Profile.Market)
	}
	if ctx.ProfileKey != "work" {
		t.Fatalf("profile key: %s", ctx.ProfileKey)
	}
}

func TestResolveCookiePath(t *testing.T) {
	ctx := &Context{ConfigPath: "/tmp/spogo/config.toml", ProfileKey: "default"}
	path := ctx.ResolveCookiePath()
	if filepath.Base(path) != "default.json" {
		t.Fatalf("cookie path: %s", path)
	}
}

func TestClearCache(t *testing.T) {
	dir := t.TempDir()
	ctx := &Context{ConfigPath: filepath.Join(dir, "config.toml"), ProfileKey: "default"}
	path := ctx.ResolveCachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := ctx.ClearCache(); err != nil {
		t.Fatalf("clear cache: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected cache removed, err=%v", err)
	}
	if err := ctx.ClearCache(); err != nil {
		t.Fatalf("clear missing cache: %v", err)
	}
}

func TestValidateProfile(t *testing.T) {
	ctx := &Context{Profile: config.Profile{Market: "USA"}}
	if err := ctx.ValidateProfile(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestEnsureTimeout(t *testing.T) {
	ctx := &Context{Settings: Settings{}}
	if ctx.EnsureTimeout() == 0 {
		t.Fatalf("expected default timeout")
	}
	ctx = &Context{Settings: Settings{Timeout: time.Second}}
	if ctx.EnsureTimeout() != time.Second {
		t.Fatalf("expected custom timeout")
	}
}

func TestSpotifyCachedClient(t *testing.T) {
	ctx := &Context{}
	ctx.SetSpotify(dummySpotify{})
	client, err := ctx.Spotify()
	if err != nil {
		t.Fatalf("spotify: %v", err)
	}
	if _, ok := client.(dummySpotify); !ok {
		t.Fatalf("expected cached client")
	}
}

func TestSaveProfile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := config.Default()
	ctx := &Context{Config: cfg, ConfigPath: path, ProfileKey: "default"}
	if err := ctx.SaveProfile(config.Profile{Market: "US"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Profile("default").Market != "US" {
		t.Fatalf("profile not saved")
	}
}

func TestSaveProfileNilContext(t *testing.T) {
	var ctx *Context
	if err := ctx.SaveProfile(config.Profile{Market: "US"}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCookieSourceFile(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json"}}
	src, err := ctx.cookieSource()
	if err != nil {
		t.Fatalf("cookie source: %v", err)
	}
	if _, ok := src.(cookies.FileSource); !ok {
		t.Fatalf("expected file source")
	}
}

func TestCookieSourceBrowser(t *testing.T) {
	ctx := &Context{Profile: config.Profile{Browser: "chrome"}}
	src, err := ctx.cookieSource()
	if err != nil {
		t.Fatalf("cookie source: %v", err)
	}
	if _, ok := src.(cookies.BrowserSource); !ok {
		t.Fatalf("expected browser source")
	}
}

func TestCookieSourceDefaultBrowser(t *testing.T) {
	ctx := &Context{Profile: config.Profile{}}
	src, err := ctx.cookieSource()
	if err != nil {
		t.Fatalf("cookie source: %v", err)
	}
	browser, ok := src.(cookies.BrowserSource)
	if !ok || browser.Browser != "chrome" {
		t.Fatalf("expected chrome source")
	}
}

func TestSpotifyNilContext(t *testing.T) {
	var ctx *Context
	if _, err := ctx.Spotify(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSpotifyBuildsClient(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json"}}
	client, err := ctx.Spotify()
	if err != nil {
		t.Fatalf("spotify: %v", err)
	}
	if client == nil {
		t.Fatalf("expected client")
	}
}

func TestSpotifyWebEngine(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json", Engine: "web"}}
	client, err := ctx.Spotify()
	if err != nil {
		t.Fatalf("spotify: %v", err)
	}
	if client == nil {
		t.Fatalf("expected client")
	}
}

func TestSpotifyAutoEngine(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json", Engine: "auto"}}
	client, err := ctx.Spotify()
	if err != nil {
		t.Fatalf("spotify: %v", err)
	}
	if client == nil {
		t.Fatalf("expected client")
	}
}

func TestSpotifyUnknownEngine(t *testing.T) {
	ctx := &Context{Profile: config.Profile{CookiePath: "/tmp/cookies.json", Engine: "nope"}}
	if _, err := ctx.Spotify(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestIsColorEnabled(t *testing.T) {
	if isColorEnabled(output.FormatJSON, false) {
		t.Fatalf("expected false")
	}
	if isColorEnabled(output.FormatHuman, true) {
		t.Fatalf("expected false")
	}
	t.Setenv("NO_COLOR", "1")
	if isColorEnabled(output.FormatHuman, false) {
		t.Fatalf("expected false")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if isColorEnabled(output.FormatHuman, false) {
		t.Fatalf("expected false")
	}
}

func TestNewContextInvalidFormat(t *testing.T) {
	_, err := NewContext(Settings{Format: "bad"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNewContextInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.toml")
	if err := os.WriteFile(path, []byte("not=toml=\""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := NewContext(Settings{ConfigPath: path, Format: output.FormatPlain}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSetSpotifyNilContext(t *testing.T) {
	var ctx *Context
	ctx.SetSpotify(dummySpotify{})
}

func TestSetSpotify(t *testing.T) {
	ctx := &Context{}
	ctx.SetSpotify(dummySpotify{})
	if ctx.spotifyClient == nil {
		t.Fatalf("expected spotify client")
	}
}

func TestCommandContextFallbacks(t *testing.T) {
	var nilCtx *Context
	if nilCtx.CommandContext() == nil {
		t.Fatalf("expected background context for nil app context")
	}

	ctx := &Context{}
	if ctx.CommandContext() == nil {
		t.Fatalf("expected background context for empty app context")
	}
}

func TestSetCommandContext(t *testing.T) {
	ctx := &Context{}
	cmdCtx, cancel := context.WithCancel(context.Background())
	ctx.SetCommandContext(cmdCtx)
	cancel()

	select {
	case <-ctx.CommandContext().Done():
	default:
		t.Fatalf("expected stored command context")
	}

	var nilContext context.Context
	ctx.SetCommandContext(nilContext)
	if ctx.CommandContext() == nil {
		t.Fatalf("expected background context after nil reset")
	}

	var nilCtx *Context
	nilCtx.SetCommandContext(cmdCtx)
}

func TestNewContextAppliesRequestedProfileAndSettings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := config.Default()
	cfg.SetProfile("work", config.Profile{
		Market:   "US",
		Language: "en",
		Device:   "speaker",
		Engine:   "web",
	})
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	ctx, err := NewContext(Settings{
		ConfigPath: path,
		Profile:    "work",
		Market:     "DE",
		Language:   "de",
		Device:     "desktop",
		Engine:     "auto",
		Format:     output.FormatPlain,
	})
	if err != nil {
		t.Fatalf("new context: %v", err)
	}
	if ctx.ProfileKey != "work" {
		t.Fatalf("profile key: %s", ctx.ProfileKey)
	}
	if ctx.Profile.Market != "DE" || ctx.Profile.Language != "de" || ctx.Profile.Device != "desktop" || ctx.Profile.Engine != "auto" {
		t.Fatalf("profile overrides not applied: %+v", ctx.Profile)
	}
}

func TestResolveProfileKeyFallbacks(t *testing.T) {
	if got := resolveProfileKey(nil, "work"); got != "work" {
		t.Fatalf("requested profile: %s", got)
	}
	if got := resolveProfileKey(&config.Config{DefaultProfile: "primary"}, ""); got != "primary" {
		t.Fatalf("default profile: %s", got)
	}
	if got := resolveProfileKey(nil, ""); got != config.DefaultProfile {
		t.Fatalf("fallback profile: %s", got)
	}
}

func TestApplySettingsKeepsEmptyValues(t *testing.T) {
	profile := applySettings(config.Profile{
		Market:   "US",
		Language: "en",
		Device:   "speaker",
		Engine:   "web",
	}, Settings{})

	if profile.Market != "US" || profile.Language != "en" || profile.Device != "speaker" || profile.Engine != "web" {
		t.Fatalf("profile changed unexpectedly: %+v", profile)
	}
}

func TestValidateProfileOK(t *testing.T) {
	ctx := &Context{Profile: config.Profile{Market: "US"}}
	if err := ctx.ValidateProfile(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type dummySpotify struct{}

func (dummySpotify) Search(context.Context, string, string, int, int) (spotify.SearchResult, error) {
	return spotify.SearchResult{}, nil
}

func (dummySpotify) GetTrack(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) GetAlbum(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) GetArtist(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) GetPlaylist(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) GetShow(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) GetEpisode(context.Context, string) (spotify.Item, error) {
	return spotify.Item{}, nil
}

func (dummySpotify) Playback(context.Context) (spotify.PlaybackStatus, error) {
	return spotify.PlaybackStatus{}, nil
}
func (dummySpotify) Play(context.Context, string) error                { return nil }
func (dummySpotify) Pause(context.Context) error                       { return nil }
func (dummySpotify) Next(context.Context) error                        { return nil }
func (dummySpotify) Previous(context.Context) error                    { return nil }
func (dummySpotify) Seek(context.Context, int) error                   { return nil }
func (dummySpotify) Volume(context.Context, int) error                 { return nil }
func (dummySpotify) Shuffle(context.Context, bool) error               { return nil }
func (dummySpotify) Repeat(context.Context, string) error              { return nil }
func (dummySpotify) Devices(context.Context) ([]spotify.Device, error) { return nil, nil }
func (dummySpotify) Transfer(context.Context, string) error            { return nil }
func (dummySpotify) QueueAdd(context.Context, string) error            { return nil }
func (dummySpotify) Queue(context.Context) (spotify.Queue, error)      { return spotify.Queue{}, nil }
func (dummySpotify) LibraryTracks(context.Context, int, int) ([]spotify.Item, int, error) {
	return nil, 0, nil
}

func (dummySpotify) LibraryAlbums(context.Context, int, int) ([]spotify.Item, int, error) {
	return nil, 0, nil
}
func (dummySpotify) LibraryModify(context.Context, string, []string, string) error { return nil }
func (dummySpotify) FollowArtists(context.Context, []string, string) error         { return nil }
func (dummySpotify) FollowedArtists(context.Context, int, string) ([]spotify.Item, int, string, error) {
	return nil, 0, "", nil
}

func (dummySpotify) Playlists(context.Context, int, int) ([]spotify.Item, int, error) {
	return nil, 0, nil
}

func (dummySpotify) PlaylistTracks(context.Context, string, int, int) ([]spotify.Item, int, error) {
	return nil, 0, nil
}

func (dummySpotify) CreatePlaylist(context.Context, string, bool, bool) (spotify.Item, error) {
	return spotify.Item{}, nil
}
func (dummySpotify) AddTracks(context.Context, string, []string) error    { return nil }
func (dummySpotify) RemoveTracks(context.Context, string, []string) error { return nil }
func (dummySpotify) GetUsersTopTracks(context.Context, string, int, int) (spotify.TopTracksResult, error) {
	return spotify.TopTracksResult{}, nil
}

func (dummySpotify) GetRecentlyPlayed(context.Context, int, int64, int64) (spotify.RecentlyPlayedResult, error) {
	return spotify.RecentlyPlayedResult{}, nil
}

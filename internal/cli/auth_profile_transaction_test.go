package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/steipete/spogo/internal/config"
	"github.com/steipete/spogo/internal/cookies"
	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestCookiePasteCannotOverwriteOrClearAnotherProfile(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "../cookies/victim"
	ctx.Config = config.Default()
	victimPath := config.CookiePath(ctx.ConfigPath, "victim")
	if err := os.MkdirAll(filepath.Dir(victimPath), 0o700); err != nil {
		t.Fatal(err)
	}
	const sentinel = "synthetic existing profile"
	if err := os.WriteFile(victimPath, []byte(sentinel), 0o600); err != nil {
		t.Fatal(err)
	}
	withStdin(t, "sp_dc=synthetic\nsp_t=synthetic\n", func() {
		if err := (&AuthPasteCmd{}).Run(ctx); err != nil {
			t.Fatal(err)
		}
	})
	if got, err := os.ReadFile(victimPath); err != nil || string(got) != sentinel {
		t.Fatalf("another profile's cookies changed: %q, %v", got, err)
	}
	stored, err := cookies.Read(ctx.ResolveCookiePath())
	if err != nil || len(stored) != 2 {
		t.Fatalf("new profile's cookies were not retained: count=%d, %v", len(stored), err)
	}
}

func TestCookiePastePreservesNewerOAuthSettings(t *testing.T) {
	for _, auth := range []string{"oauth", ""} {
		t.Run("persisted_"+auth, func(t *testing.T) {
			ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
			ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
			ctx.ProfileKey = "default"
			ctx.Config = config.Default()
			// The command started before another command changed auth and preferences.
			ctx.Profile = config.Profile{Auth: "oauth", SpotifyClientID: "old-client", Browser: "chrome"}
			current := config.Profile{Auth: auth, SpotifyClientID: "new-client", SpotifyRedirectURI: "http://127.0.0.1:8888/callback", Browser: "firefox", Market: "AT", Language: "de", Device: "speaker", Engine: "web"}
			ctx.Config.SetProfile("default", current)
			if err := config.Save(ctx.ConfigPath, ctx.Config); err != nil {
				t.Fatal(err)
			}
			withStdin(t, "sp_dc=synthetic\nsp_t=synthetic\n", func() {
				if err := (&AuthPasteCmd{}).Run(ctx); err != nil {
					t.Fatal(err)
				}
			})
			loaded, err := config.Load(ctx.ConfigPath)
			if err != nil {
				t.Fatal(err)
			}
			current.CookiePath = ctx.ResolveCookiePath()
			if got := loaded.Profile("default"); got != current {
				t.Fatalf("lost current settings: got %+v want %+v", got, current)
			}
		})
	}
}

func TestOAuthClearWaitsForConfigTransaction(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	ctx.Config = config.Default()
	ctx.Config.SetProfile("default", config.Profile{Auth: "oauth", SpotifyClientID: "client"})
	if err := config.Save(ctx.ConfigPath, ctx.Config); err != nil {
		t.Fatal(err)
	}
	path := ctx.ResolveOAuthTokenPath()
	if err := spotify.SaveOAuthToken(path, spotify.OAuthToken{AccessToken: "synthetic", RefreshToken: "synthetic", ClientID: "client", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	lock := flock.New(ctx.ConfigPath + ".lock")
	if err := lock.Lock(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Close() }()
	// Model another writer between truncation and completion while holding its lock.
	if err := os.WriteFile(ctx.ConfigPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	commandCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	ctx.SetCommandContext(commandCtx)
	if err := (&AuthOAuthClearCmd{}).Run(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("clear did not wait for config: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("clear removed token before config commit: %v", err)
	}
	if err := config.Save(ctx.ConfigPath, ctx.Config); err != nil {
		t.Fatal(err)
	}
	if err := lock.Unlock(); err != nil {
		t.Fatal(err)
	}
	ctx.SetCommandContext(context.Background())
	if err := (&AuthOAuthClearCmd{}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(ctx.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Profile("default").Auth != "" {
		t.Fatal("OAuth still selected")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("token still present: %v", err)
	}
}

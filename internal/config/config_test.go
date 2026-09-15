package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func isolateConfigHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	return dir
}

func TestLoadDefaultWhenMissing(t *testing.T) {
	isolateConfigHome(t)
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("load default: %v", err)
	}
	if cfg.DefaultProfile != DefaultProfile {
		t.Fatalf("default profile = %q", cfg.DefaultProfile)
	}
}

func TestDefaultPath(t *testing.T) {
	isolateConfigHome(t)
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}
	if filepath.Base(path) != DefaultConfig {
		t.Fatalf("unexpected path: %s", path)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := Default()
	cfg.DefaultProfile = "work"
	cfg.SetProfile("work", Profile{Browser: "chrome", Market: "US"})
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	profile := loaded.Profile("work")
	if profile.Browser != "chrome" || profile.Market != "US" {
		t.Fatalf("profile mismatch: %#v", profile)
	}
}

func TestSaveLoadOAuthSettingsWithoutClientSecret(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := Default()
	cfg.SetProfile("default", Profile{
		Auth:               "oauth",
		SpotifyClientID:    "client-id",
		SpotifyRedirectURI: "http://127.0.0.1:8888/callback",
	})
	if err := Save(path, cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) == "" || !strings.Contains(string(data), "spotify_client_id") {
		t.Fatalf("oauth settings missing: %s", data)
	}
	if strings.Contains(strings.ToLower(string(data)), "client_secret") {
		t.Fatalf("config must not contain a client secret field: %s", data)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := loaded.Profile("default"); got.Auth != "oauth" || got.SpotifyClientID != "client-id" {
		t.Fatalf("oauth profile mismatch: %+v", got)
	}
}

func TestCookiePath(t *testing.T) {
	path := CookiePath("/tmp/spogo/config.toml", "default")
	if filepath.Base(path) != "default.json" {
		t.Fatalf("cookie path: %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}

func TestCookiePathEmptyConfig(t *testing.T) {
	if CookiePath("", "default") != "" {
		t.Fatalf("expected empty")
	}
}

func TestCachePath(t *testing.T) {
	path := CachePath("/tmp/spogo/config.toml", "default")
	if filepath.Base(path) != "default.json" {
		t.Fatalf("cache path: %s", path)
	}
	if filepath.Base(filepath.Dir(path)) != "cache" {
		t.Fatalf("cache dir: %s", path)
	}
}

func TestCachePathEmptyConfig(t *testing.T) {
	if CachePath("", "default") != "" {
		t.Fatalf("expected empty")
	}
}

func TestOAuthTokenPath(t *testing.T) {
	path := OAuthTokenPath("/tmp/spogo/config.toml", "work")
	if filepath.Base(path) != "work.json" || filepath.Base(filepath.Dir(path)) != "oauth" {
		t.Fatalf("oauth token path: %s", path)
	}
	if OAuthTokenPath("", "default") != "" {
		t.Fatalf("expected empty oauth token path")
	}
}

func TestProfilePathsContainUnsafeProfiles(t *testing.T) {
	for _, tc := range []struct {
		dir  string
		path func(string, string) string
	}{
		{"cookies", CookiePath},
		{"cache", CachePath},
		{"oauth", OAuthTokenPath},
	} {
		t.Run(tc.dir, func(t *testing.T) {
			testProfilePathsContainUnsafeProfiles(t, tc.dir, tc.path)
		})
	}
}

func testProfilePathsContainUnsafeProfiles(t *testing.T, directory string, pathFor func(string, string) string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "spogo", "config.toml")
	profileDir := filepath.Join(filepath.Dir(configPath), directory)
	unsafeProfiles := []string{
		"../other",
		"../cookies/victim",
		"work/personal",
		`work\personal`,
		".",
		"..",
		"profile.",
		"CON",
		"com1.txt",
		"personal account",
		"00a ",
		"00G ",
		"WORK",
		"Work",
		"~576f726b",
		strings.Repeat("A", 125),
		strings.Repeat("a", 251),
		strings.Repeat("日本語", 50),
	}
	seen := map[string]string{}
	for _, profile := range unsafeProfiles {
		path := pathFor(configPath, profile)
		if filepath.Dir(path) != profileDir {
			t.Fatalf("profile %q escaped %s directory: %s", profile, directory, path)
		}
		name := filepath.Base(path)
		if len(name+".lifecycle.lock") > 255 {
			t.Fatalf("profile %q exceeds the filesystem filename limit: %s", profile, name)
		}
		if !strings.HasPrefix(name, "~") || filepath.Ext(name) != ".json" {
			t.Fatalf("profile %q was not safely encoded: %s", profile, name)
		}
		if strings.ContainsAny(name, `/\`) {
			t.Fatalf("profile %q retained a path separator: %s", profile, name)
		}
		collisionKey := strings.ToLower(name)
		if previous, ok := seen[collisionKey]; ok {
			t.Fatalf("profiles %q and %q collided at %s", previous, profile, name)
		}
		seen[collisionKey] = profile
	}
	if got := filepath.Base(pathFor(configPath, "work.prod-1")); got != "work.prod-1.json" {
		t.Fatalf("portable profile path changed: %s", got)
	}
	if pathFor(configPath, "") != pathFor(configPath, DefaultProfile) {
		t.Fatal("empty profile must select the default profile")
	}
}

func TestLoadInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte("not=toml=\""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadReadError(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSaveNilConfig(t *testing.T) {
	if err := Save("", nil); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSaveDefaultPath(t *testing.T) {
	isolateConfigHome(t)
	cfg := Default()
	if err := Save("", cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
}

func TestSaveInvalidDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	path := filepath.Join(file, "config.toml")
	if err := Save(path, Default()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpdateErrorsAndDefaultPath(t *testing.T) {
	if _, err := Update(context.Background(), filepath.Join(t.TempDir(), "config.toml"), nil); err == nil {
		t.Fatal("expected nil update error")
	}

	isolateConfigHome(t)
	wantErr := context.Canceled
	if _, err := Update(context.Background(), "", func(*Config) error { return wantErr }); !errors.Is(err, wantErr) {
		t.Fatalf("callback error = %v, want %v", err, wantErr)
	}
	updated, err := Update(context.Background(), "", func(cfg *Config) error {
		cfg.SetProfile("default", Profile{Market: "US"})
		return nil
	})
	if err != nil {
		t.Fatalf("default-path update: %v", err)
	}
	if got := updated.Profile("default").Market; got != "US" {
		t.Fatalf("updated market = %q", got)
	}
}

func TestUpdateLoadAndSaveErrors(t *testing.T) {
	dir := t.TempDir()
	invalidPath := filepath.Join(dir, "invalid.toml")
	if err := os.WriteFile(invalidPath, []byte("not=toml=\""), 0o644); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	if _, err := Update(context.Background(), invalidPath, func(*Config) error { return nil }); err == nil {
		t.Fatal("expected load error")
	}

	savePath := filepath.Join(dir, "save-error.toml")
	if _, err := Update(context.Background(), savePath, func(*Config) error {
		if err := os.Mkdir(savePath, 0o755); err != nil {
			return err
		}
		return nil
	}); err == nil {
		t.Fatal("expected save error")
	}
}

func TestUpdateHonorsLockCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		_, err := Update(context.Background(), path, func(*Config) error {
			close(firstEntered)
			<-releaseFirst
			return nil
		})
		firstDone <- err
	}()
	<-firstEntered

	waitCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := Update(waitCtx, path, func(*Config) error { return nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock wait error = %v", err)
	}
	close(releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatalf("first update: %v", err)
	}
}

func TestUpdateSerializesDifferentProfileWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := Save(path, Default()); err != nil {
		t.Fatalf("save initial config: %v", err)
	}

	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		_, err := Update(context.Background(), path, func(cfg *Config) error {
			cfg.SetProfile("personal", Profile{Auth: "oauth", SpotifyClientID: "personal-client"})
			close(firstEntered)
			<-releaseFirst
			return nil
		})
		firstDone <- err
	}()
	<-firstEntered

	secondEntered := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		_, err := Update(context.Background(), path, func(cfg *Config) error {
			close(secondEntered)
			cfg.SetProfile("work", Profile{Auth: "oauth", SpotifyClientID: "work-client"})
			return nil
		})
		secondDone <- err
	}()
	select {
	case <-secondEntered:
		t.Fatal("second profile update bypassed the config lock")
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatalf("first update: %v", err)
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second update: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load final config: %v", err)
	}
	if got := loaded.Profile("personal").SpotifyClientID; got != "personal-client" {
		t.Fatalf("personal profile lost: %q", got)
	}
	if got := loaded.Profile("work").SpotifyClientID; got != "work-client" {
		t.Fatalf("work profile lost: %q", got)
	}
}

func TestProfileNilConfig(t *testing.T) {
	var cfg *Config
	if p := cfg.Profile("default"); p != (Profile{}) {
		t.Fatalf("expected empty profile")
	}
}

func TestSetProfileDefaultName(t *testing.T) {
	cfg := Default()
	cfg.SetProfile("", Profile{Market: "US"})
	if cfg.Profile(DefaultProfile).Market != "US" {
		t.Fatalf("expected profile set")
	}
}

func TestSetProfileNilMap(t *testing.T) {
	cfg := &Config{}
	cfg.SetProfile("", Profile{Market: "DE"})
	if cfg.Profiles == nil {
		t.Fatalf("expected profiles map")
	}
	if cfg.Profile(DefaultProfile).Market != "DE" {
		t.Fatalf("expected profile")
	}
}

func TestSetProfileNilConfig(t *testing.T) {
	var cfg *Config
	cfg.SetProfile("default", Profile{Market: "US"})
}

func TestProfileNilMap(t *testing.T) {
	cfg := &Config{DefaultProfile: DefaultProfile}
	if cfg.Profile("default") != (Profile{}) {
		t.Fatalf("expected empty profile")
	}
}

func TestProfileFallback(t *testing.T) {
	cfg := Default()
	cfg.DefaultProfile = "primary"
	cfg.SetProfile("primary", Profile{Market: "DE"})
	if cfg.Profile("").Market != "DE" {
		t.Fatalf("expected default profile")
	}
}

func TestNormalize(t *testing.T) {
	cfg := &Config{}
	cfg.normalize()
	if cfg.DefaultProfile == "" || cfg.Profiles == nil {
		t.Fatalf("expected defaults")
	}
}

func TestSavePreservesConfigSymlink(t *testing.T) {
	for _, existing := range []bool{true, false} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			dir := t.TempDir()
			target := filepath.Join(dir, "managed.toml")
			link := filepath.Join(dir, "config.toml")
			if existing {
				if err := Save(target, Default()); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink("managed.toml", link); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			cfg := Default()
			cfg.SetProfile("default", Profile{Market: "AT"})
			if err := Save(link, cfg); err != nil {
				t.Fatal(err)
			}
			if destination, err := os.Readlink(link); err != nil || destination != "managed.toml" {
				t.Fatalf("config link replaced: %q %v", destination, err)
			}
			loaded, err := Load(target)
			if err != nil || loaded.Profile("default").Market != "AT" {
				t.Fatalf("target was not updated: %v", err)
			}
		})
	}
}

func TestUpdateConfigAliasesShareLock(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "managed.toml")
	link := filepath.Join(dir, "config.toml")
	if err := Save(target, Default()); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := Update(context.Background(), target, func(*Config) error {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, err := Update(ctx, link, func(*Config) error { t.Error("alias bypassed config lock"); return nil })
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("alias lock error: %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSaveRejectsCyclicConfigSymlinks(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}
	if err := Save(a, Default()); err == nil {
		t.Fatal("expected symlink cycle error")
	}
}

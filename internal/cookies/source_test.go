package cookies

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/steipete/sweetcookie"
)

func TestBrowserSource(t *testing.T) {
	exp := time.Now().Add(time.Hour)
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		return sweetcookie.Result{
			Cookies: []sweetcookie.Cookie{{
				Name:     "sp_dc",
				Value:    "token",
				Domain:   ".spotify.com",
				Path:     "/",
				Secure:   true,
				HTTPOnly: true,
				Expires:  &exp,
			}},
		}, nil
	})
	defer restore()
	src := BrowserSource{Browser: "chrome", Profile: "Default", Domain: "spotify.com"}
	cookies, err := src.Cookies(context.Background())
	if err != nil {
		t.Fatalf("cookies: %v", err)
	}
	if len(cookies) != 1 || cookies[0].Name != "sp_dc" {
		t.Fatalf("unexpected cookies: %#v", cookies)
	}
	if cookies[0].Path != "/" || !cookies[0].Secure || !cookies[0].HttpOnly || !cookies[0].Expires.Equal(exp) {
		t.Fatalf("unexpected cookie fields: %#v", cookies[0])
	}
}

func TestBrowserSourceCopiesNilExpiry(t *testing.T) {
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		return sweetcookie.Result{
			Cookies: []sweetcookie.Cookie{{Name: "sp_dc", Value: "token", Domain: ".spotify.com"}},
		}, nil
	})
	defer restore()
	src := BrowserSource{Browser: "firefox", Domain: "spotify.com"}
	cookies, err := src.Cookies(context.Background())
	if err != nil {
		t.Fatalf("cookies: %v", err)
	}
	if len(cookies) != 1 || !cookies[0].Expires.IsZero() {
		t.Fatalf("unexpected cookies: %#v", cookies)
	}
}

func TestBrowserSourceNoCookies(t *testing.T) {
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		return sweetcookie.Result{}, nil
	})
	defer restore()
	src := BrowserSource{Domain: "spotify.com"}
	if _, err := src.Cookies(context.Background()); !errors.Is(err, ErrNoCookies) {
		t.Fatalf("expected ErrNoCookies, got %v", err)
	}
}

func TestBrowserSourceRetriesAcrossSpotifyHosts(t *testing.T) {
	var calls []sweetcookie.Options
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		calls = append(calls, opts)
		if len(calls) == 1 {
			return sweetcookie.Result{}, nil
		}
		return sweetcookie.Result{
			Cookies: []sweetcookie.Cookie{{Name: "sp_dc", Value: "token", Domain: ".accounts.spotify.com"}},
		}, nil
	})
	defer restore()
	src := BrowserSource{Browser: "chrome", Domain: "spotify.com"}
	cookies, err := src.Cookies(context.Background())
	if err != nil {
		t.Fatalf("cookies: %v", err)
	}
	if len(cookies) != 1 || cookies[0].Name != "sp_dc" {
		t.Fatalf("unexpected cookies: %#v", cookies)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0].AllowAllHosts {
		t.Fatalf("expected filtered first pass")
	}
	if !calls[1].AllowAllHosts {
		t.Fatalf("expected allow-all fallback")
	}
	if len(calls[1].Names) != len(authCookieNames) {
		t.Fatalf("expected auth cookie allowlist")
	}
}

func TestBrowserSourceRetryError(t *testing.T) {
	calls := 0
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		calls++
		if calls == 1 {
			return sweetcookie.Result{}, nil
		}
		return sweetcookie.Result{}, errors.New("retry boom")
	})
	defer restore()
	src := BrowserSource{Browser: "firefox", Domain: "spotify.com"}
	if _, err := src.Cookies(context.Background()); err == nil || !strings.Contains(err.Error(), "retry boom") {
		t.Fatalf("expected retry error, got %v", err)
	}
}

func TestSetReadCookies(t *testing.T) {
	restore := SetReadCookies(nil)
	restore()
	restore = SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		return sweetcookie.Result{}, nil
	})
	restore()
}

func TestBrowserSourceError(t *testing.T) {
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		return sweetcookie.Result{}, errors.New("boom")
	})
	defer restore()
	src := BrowserSource{Domain: "spotify.com"}
	if _, err := src.Cookies(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestBrowserSourceNoCookiesIncludesWarnings(t *testing.T) {
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		return sweetcookie.Result{Warnings: []string{"sweetcookie: chrome cookie store not found"}}, nil
	})
	defer restore()
	src := BrowserSource{Browser: "chrome", Domain: "spotify.com"}
	_, err := src.Cookies(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "chrome cookie store not found") {
		t.Fatalf("expected warning in error, got %v", err)
	}
	if !errors.Is(err, ErrNoCookies) {
		t.Fatalf("expected ErrNoCookies, got %v", err)
	}
}

func TestBrowserSourceUsesSpotifyOrigins(t *testing.T) {
	var got sweetcookie.Options
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		got = opts
		return sweetcookie.Result{
			Cookies: []sweetcookie.Cookie{{Name: "sp_dc", Value: "token", Domain: ".spotify.com"}},
		}, nil
	})
	defer restore()
	src := BrowserSource{Browser: "chrome", Domain: "spotify.com"}
	if _, err := src.Cookies(context.Background()); err != nil {
		t.Fatalf("expected cookies: %v", err)
	}
	if len(got.Origins) != 2 {
		t.Fatalf("expected 2 spotify origins, got %v", got.Origins)
	}
	if got.Origins[0] != "https://open.spotify.com" && got.Origins[1] != "https://open.spotify.com" {
		t.Fatalf("expected open.spotify.com origin, got %v", got.Origins)
	}
	if got.Origins[0] != "https://accounts.spotify.com" && got.Origins[1] != "https://accounts.spotify.com" {
		t.Fatalf("expected accounts.spotify.com origin, got %v", got.Origins)
	}
}

func TestBrowserSourceURLDomain(t *testing.T) {
	var got sweetcookie.Options
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		got = opts
		return sweetcookie.Result{
			Cookies: []sweetcookie.Cookie{{Name: "sp_dc", Value: "token", Domain: ".open.spotify.com"}},
		}, nil
	})
	defer restore()
	src := BrowserSource{Browser: "firefox", Domain: "https://open.spotify.com/collection"}
	if _, err := src.Cookies(context.Background()); err != nil {
		t.Fatalf("expected cookies: %v", err)
	}
	if got.URL != "https://open.spotify.com" {
		t.Fatalf("expected parsed domain URL, got %q", got.URL)
	}
	if len(got.Origins) != 2 {
		t.Fatalf("expected related spotify origins, got %v", got.Origins)
	}
}

func TestSpotifyOriginsIgnoresEmptyAndNonSpotifyHosts(t *testing.T) {
	for _, host := range []string{"", "example.com"} {
		if got := spotifyOrigins(host); got != nil {
			t.Fatalf("expected no origins for %q, got %v", host, got)
		}
	}
}

func TestCompactWarningsTrimsDeduplicatesAndLimits(t *testing.T) {
	got := compactWarnings([]string{" first ", "", "first", "second", "third", "fourth"})
	want := []string{"first", "second", "third"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
	if got := compactWarnings([]string{"", "   "}); got != nil {
		t.Fatalf("expected nil warnings, got %v", got)
	}
}

func TestBrowserSourceDefaultDomain(t *testing.T) {
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		return sweetcookie.Result{
			Cookies: []sweetcookie.Cookie{{Name: "sp_dc", Value: "token", Domain: ".spotify.com"}},
		}, nil
	})
	defer restore()
	src := BrowserSource{}
	if _, err := src.Cookies(context.Background()); err != nil {
		t.Fatalf("expected cookies")
	}
}

func TestBrowserSourceWithProfileFilter(t *testing.T) {
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		return sweetcookie.Result{
			Cookies: []sweetcookie.Cookie{{Name: "sp_dc", Value: "token", Domain: ".spotify.com"}},
		}, nil
	})
	defer restore()
	src := BrowserSource{Browser: "chrome", Profile: "Default", Domain: "spotify.com"}
	cookies, err := src.Cookies(context.Background())
	if err != nil || len(cookies) != 1 {
		t.Fatalf("expected cookie")
	}
}

func TestBrowserSourceProfileOnlyUsesDefaults(t *testing.T) {
	var got sweetcookie.Options
	restore := SetReadCookies(func(ctx context.Context, opts sweetcookie.Options) (sweetcookie.Result, error) {
		got = opts
		return sweetcookie.Result{
			Cookies: []sweetcookie.Cookie{{Name: "sp_dc", Value: "token", Domain: ".spotify.com"}},
		}, nil
	})
	defer restore()
	src := BrowserSource{Profile: "Default"}
	if _, err := src.Cookies(context.Background()); err != nil {
		t.Fatalf("expected cookies")
	}
	if len(got.Profiles) == 0 {
		t.Fatalf("expected default profiles map")
	}
}

func TestFileSource(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/cookies.json"
	if err := Write(path, []*http.Cookie{{Name: "sp_dc", Value: "token"}}); err != nil {
		t.Fatalf("write: %v", err)
	}
	src := FileSource{Path: path}
	cookies, err := src.Cookies(context.Background())
	if err != nil {
		t.Fatalf("cookies: %v", err)
	}
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie")
	}
}

func TestFileSourceError(t *testing.T) {
	src := FileSource{Path: "/nope/missing.json"}
	if _, err := src.Cookies(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

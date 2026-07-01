package cookies

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/steipete/sweetcookie"
)

var readCookies = sweetcookie.Get

var authCookieNames = []string{"sp_dc", "sp_key", "sp_t"}

// ErrNoCookies reports that all configured browser cookie sources were empty.
var ErrNoCookies = errors.New("no cookies found")

// SetReadCookies overrides the internal cookie reader and returns a restore func.
// Intended for tests.
func SetReadCookies(fn func(context.Context, sweetcookie.Options) (sweetcookie.Result, error)) func() {
	prev := readCookies
	if fn == nil {
		readCookies = sweetcookie.Get
	} else {
		readCookies = fn
	}
	return func() { readCookies = prev }
}

type Source interface {
	Cookies(ctx context.Context) ([]*http.Cookie, error)
}

type BrowserSource struct {
	Browser string
	Profile string
	Domain  string
}

type FileSource struct {
	Path string
}

func (s BrowserSource) Cookies(ctx context.Context) ([]*http.Cookie, error) {
	result, err := readCookies(ctx, s.cookieOptions(false))
	if err != nil {
		return nil, err
	}
	if len(result.Cookies) == 0 && s.shouldRetryAcrossHosts() {
		retry, retryErr := readCookies(ctx, s.cookieOptions(true))
		if retryErr != nil {
			return nil, retryErr
		}
		result.Cookies = retry.Cookies
		result.Warnings = append(result.Warnings, retry.Warnings...)
	}
	if len(result.Cookies) == 0 {
		return nil, browserCookiesNotFoundError(result.Warnings)
	}
	ret := make([]*http.Cookie, 0, len(result.Cookies))
	for _, c := range result.Cookies {
		cookie := &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
		}
		if c.Expires != nil {
			cookie.Expires = *c.Expires
		}
		ret = append(ret, cookie)
	}
	return ret, nil
}

func (s BrowserSource) cookieOptions(allowAllHosts bool) sweetcookie.Options {
	host := s.host()
	opts := sweetcookie.Options{
		Mode:    sweetcookie.ModeFirst,
		Timeout: 5 * time.Second,
		Names:   authCookieNames,
	}
	if allowAllHosts {
		opts.AllowAllHosts = true
	} else {
		opts.URL = "https://" + host
		opts.Origins = spotifyOrigins(host)
	}
	if s.Browser != "" {
		browser := sweetcookie.Browser(strings.ToLower(s.Browser))
		opts.Browsers = []sweetcookie.Browser{browser}
		if s.Profile != "" {
			opts.Profiles = map[sweetcookie.Browser]string{browser: s.Profile}
		}
	} else if s.Profile != "" {
		opts.Profiles = map[sweetcookie.Browser]string{}
		for _, browser := range sweetcookie.DefaultBrowsers() {
			opts.Profiles[browser] = s.Profile
		}
	}
	return opts
}

func (s BrowserSource) host() string {
	domain := strings.TrimSpace(s.Domain)
	if domain == "" {
		domain = "spotify.com"
	}
	if strings.Contains(domain, "://") {
		if parsed, err := url.Parse(domain); err == nil && parsed.Hostname() != "" {
			return parsed.Hostname()
		}
	}
	return domain
}

func (s BrowserSource) shouldRetryAcrossHosts() bool {
	host := normalizeCookieHost(s.host())
	return host == "spotify.com" || strings.HasSuffix(host, ".spotify.com")
}

func spotifyOrigins(host string) []string {
	host = normalizeCookieHost(host)
	if host == "" {
		return nil
	}
	if !strings.Contains(host, "spotify.com") {
		return nil
	}
	origins := []string{}
	for _, candidate := range []string{"spotify.com", "open.spotify.com", "accounts.spotify.com"} {
		if host == candidate {
			continue
		}
		origins = append(origins, "https://"+candidate)
	}
	return origins
}

func normalizeCookieHost(host string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(host), "."))
}

func browserCookiesNotFoundError(warnings []string) error {
	warnings = compactWarnings(warnings)
	if len(warnings) == 0 {
		return ErrNoCookies
	}
	return fmt.Errorf("%w; %s", ErrNoCookies, strings.Join(warnings, "; "))
}

func compactWarnings(warnings []string) []string {
	out := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		out = append(out, warning)
	}
	if len(out) == 0 {
		return nil
	}
	out = slices.Compact(out)
	if len(out) > 3 {
		return out[:3]
	}
	return out
}

func (s FileSource) Cookies(ctx context.Context) ([]*http.Cookie, error) {
	_ = ctx
	return Read(s.Path)
}

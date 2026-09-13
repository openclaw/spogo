package main

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/steipete/spogo/internal/config"
	"github.com/steipete/spogo/internal/spotify"
)

type argumentTransport func(*http.Request) (*http.Response, error)

func (f argumentTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRunSearchPreservesLiteralArguments(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		query string
	}{
		{"literal", []string{"search", "track", "--", "--no-input"}, "--no-input"},
		{"flag before", []string{"--no-input", "search", "track", "query"}, "query"},
		{"flag after", []string{"search", "track", "query", "--no-input"}, "query"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.toml")
			cfg := config.Default()
			cfg.SetProfile("default", config.Profile{Engine: "web", Auth: "oauth", SpotifyClientID: "synthetic-client"})
			if err := config.Save(configPath, cfg); err != nil {
				t.Fatal(err)
			}
			if err := spotify.SaveOAuthToken(config.OAuthTokenPath(configPath, "default"), spotify.OAuthToken{AccessToken: "synthetic-access", RefreshToken: "synthetic-refresh", ClientID: "synthetic-client", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}
			previous := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = previous })
			calls := 0
			http.DefaultTransport = argumentTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/v1/search" || r.URL.Query().Get("q") != tc.query {
					t.Errorf("unexpected request %s", r.URL)
				}
				body := `{"tracks":{"items":[{"id":"t1","name":"Synthetic Track"}],"total":1,"limit":1,"offset":0}}`
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body)), Request: r}, nil
			})
			out, errOut := new(bytes.Buffer), new(bytes.Buffer)
			args := append([]string{"--config", configPath, "--json"}, tc.args...)
			if code := run(args, out, errOut); code != 0 {
				t.Fatalf("exit %d: %s", code, errOut)
			}
			if calls != 1 {
				t.Fatalf("requests = %d, want 1", calls)
			}
			if !strings.Contains(out.String(), "Synthetic Track") {
				t.Fatalf("command output bypassed supplied writer: %q", out)
			}
		})
	}
}

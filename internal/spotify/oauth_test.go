package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGenerateOAuthPKCE(t *testing.T) {
	verifier, challenge, err := GenerateOAuthPKCE()
	if err != nil {
		t.Fatalf("generate PKCE: %v", err)
	}
	if len(verifier) < 43 || len(verifier) > 128 {
		t.Fatalf("verifier length = %d", len(verifier))
	}
	if verifier == challenge || strings.ContainsAny(verifier+challenge, "+/=") {
		t.Fatalf("invalid URL-safe PKCE values")
	}
	second, _, err := GenerateOAuthPKCE()
	if err != nil {
		t.Fatalf("generate second PKCE: %v", err)
	}
	if second == verifier {
		t.Fatalf("expected unique verifier")
	}
}

func TestOAuthAuthorizationURL(t *testing.T) {
	provider, err := NewOAuthTokenProvider(OAuthOptions{
		ClientID:    "client-id",
		RedirectURI: "http://127.0.0.1:8888/callback",
		CachePath:   filepath.Join(t.TempDir(), "token.json"),
		AccountsURL: "https://accounts.example",
		Scopes:      []string{"scope-b", "scope-a"},
	})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	raw, err := provider.AuthorizationURL("state", "challenge")
	if err != nil {
		t.Fatalf("authorization URL: %v", err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	query := parsed.Query()
	if parsed.Path != "/authorize" || query.Get("response_type") != "code" || query.Get("code_challenge_method") != "S256" {
		t.Fatalf("unexpected authorization URL: %s", raw)
	}
	if query.Get("client_id") != "client-id" || query.Get("redirect_uri") != "http://127.0.0.1:8888/callback" {
		t.Fatalf("missing client settings: %s", raw)
	}
	if query.Get("scope") != "scope-b scope-a" || query.Get("state") != "state" {
		t.Fatalf("unexpected scope/state: %s", raw)
	}
}

func TestValidateOAuthRedirectURI(t *testing.T) {
	valid := []string{
		"http://127.0.0.1:8888/callback",
		"http://[::1]:8888/callback",
	}
	for _, raw := range valid {
		if err := ValidateOAuthRedirectURI(raw); err != nil {
			t.Errorf("ValidateOAuthRedirectURI(%q): %v", raw, err)
		}
	}
	invalid := []string{
		"http://localhost:8888/callback",
		"https://127.0.0.1:8888/callback",
		"http://127.0.0.1/callback",
		"http://192.168.1.2:8888/callback",
		"http://127.0.0.1:8888/callback?token=x",
	}
	for _, raw := range invalid {
		if err := ValidateOAuthRedirectURI(raw); err == nil {
			t.Errorf("expected %q to be rejected", raw)
		}
	}
}

func TestOAuthExchangeAndRefresh(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("PKCE request must not send client secret authorization: %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		switch requests.Add(1) {
		case 1:
			if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code_verifier") != "verifier" {
				t.Fatalf("unexpected exchange form: %v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "access-one",
				"refresh_token": "refresh-one",
				"token_type":    "Bearer",
				"scope":         "user-library-read",
				"expires_in":    3600,
			})
		case 2:
			if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh-one" {
				t.Fatalf("unexpected refresh form: %v", r.Form)
			}
			if r.Form.Get("client_id") != "client-id" {
				t.Fatalf("missing PKCE client ID: %v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "access-two",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		default:
			t.Fatalf("unexpected token request")
		}
	}))
	defer server.Close()

	now := time.Date(2026, 8, 27, 20, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "oauth", "default.json")
	provider, err := NewOAuthTokenProvider(OAuthOptions{
		ClientID:    "client-id",
		RedirectURI: "http://127.0.0.1:8888/callback",
		CachePath:   path,
		AccountsURL: server.URL,
		Now:         func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	stored, err := provider.ExchangeCode(context.Background(), "code", "verifier")
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if stored.AccessToken != "access-one" || stored.RefreshToken != "refresh-one" {
		t.Fatalf("unexpected exchange token: %+v", stored)
	}
	cached, err := provider.Token(context.Background())
	if err != nil {
		t.Fatalf("cached token: %v", err)
	}
	if cached.AccessToken != "access-one" || requests.Load() != 1 {
		t.Fatalf("expected cached access token, got %+v with %d requests", cached, requests.Load())
	}

	now = now.Add(2 * time.Hour)
	refreshed, err := provider.Token(context.Background())
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.AccessToken != "access-two" || requests.Load() != 2 {
		t.Fatalf("unexpected refreshed token: %+v with %d requests", refreshed, requests.Load())
	}
	persisted, err := LoadOAuthToken(path)
	if err != nil {
		t.Fatalf("load refreshed: %v", err)
	}
	if persisted.RefreshToken != "refresh-one" {
		t.Fatalf("refresh token rotation fallback failed: %+v", persisted)
	}
	if persisted.Scope != "user-library-read" {
		t.Fatalf("refresh scope fallback failed: %+v", persisted)
	}
	if persisted.TokenType != "Bearer" {
		t.Fatalf("refresh token type fallback failed: %+v", persisted)
	}
}

func TestOAuthRefreshIsLockedAcrossProviders(t *testing.T) {
	var requests atomic.Int32
	releaseRefresh := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) != 1 {
			t.Errorf("expected one refresh request")
		}
		<-releaseRefresh
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "fresh-access",
			"refresh_token": "fresh-refresh",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "oauth", "default.json")
	if err := SaveOAuthToken(path, OAuthToken{
		AccessToken:  "expired-access",
		RefreshToken: "initial-refresh",
		ExpiresAt:    time.Now().Add(-time.Hour),
		ClientID:     "client-id",
	}); err != nil {
		t.Fatalf("save expired token: %v", err)
	}
	newProvider := func() *OAuthTokenProvider {
		provider, err := NewOAuthTokenProvider(OAuthOptions{
			ClientID:    "client-id",
			CachePath:   path,
			AccountsURL: server.URL,
		})
		if err != nil {
			t.Fatalf("provider: %v", err)
		}
		return provider
	}
	providers := []*OAuthTokenProvider{newProvider(), newProvider()}
	results := make(chan Token, len(providers))
	errorsCh := make(chan error, len(providers))
	var started sync.WaitGroup
	started.Add(len(providers))
	for _, provider := range providers {
		go func(provider *OAuthTokenProvider) {
			started.Done()
			token, err := provider.Token(context.Background())
			if err != nil {
				errorsCh <- err
				return
			}
			results <- token
		}(provider)
	}
	started.Wait()
	close(releaseRefresh)
	for range providers {
		select {
		case err := <-errorsCh:
			t.Fatalf("token: %v", err)
		case token := <-results:
			if token.AccessToken != "fresh-access" {
				t.Fatalf("access token = %q", token.AccessToken)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for providers")
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("refresh requests = %d, want 1", requests.Load())
	}
}

func TestOAuthCacheLockHonorsContextCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth", "default.json")
	first, err := acquireOAuthCacheLock(context.Background(), path)
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	defer releaseOAuthCacheLock(first)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := acquireOAuthCacheLock(ctx, path); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	releaseOAuthCacheLock(nil)
}

func TestOAuthProviderPropagatesCacheLockCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth", "default.json")
	cacheLock, err := acquireOAuthCacheLock(context.Background(), path)
	if err != nil {
		t.Fatalf("hold lock: %v", err)
	}
	defer releaseOAuthCacheLock(cacheLock)
	provider, err := NewOAuthTokenProvider(OAuthOptions{
		ClientID:    "client-id",
		RedirectURI: "http://127.0.0.1:8888/callback",
		CachePath:   path,
	})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Token(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("token cancellation = %v", err)
	}
	if _, err := provider.ExchangeCode(ctx, "code", "verifier"); !errors.Is(err, context.Canceled) {
		t.Fatalf("exchange cancellation = %v", err)
	}
}

func TestOAuthProviderMissingAndMismatchedCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	provider, err := NewOAuthTokenProvider(OAuthOptions{ClientID: "client-id", CachePath: path})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err := provider.Token(context.Background()); !errors.Is(err, ErrOAuthAuthentication) {
		t.Fatalf("expected oauth auth error, got %v", err)
	}
	if err := SaveOAuthToken(path, OAuthToken{RefreshToken: "refresh", ClientID: "other"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := provider.Token(context.Background()); !errors.Is(err, ErrOAuthAuthentication) {
		t.Fatalf("expected client mismatch auth error, got %v", err)
	}
}

func TestLoadOAuthTokenMissing(t *testing.T) {
	_, err := LoadOAuthToken(filepath.Join(t.TempDir(), "missing.json"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected missing token cache, got %v", err)
	}
}

func TestOAuthTokenCachePermissionsStatusAndClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth", "default.json")
	token := OAuthToken{
		AccessToken:  "access",
		RefreshToken: "refresh",
		Scope:        "scope-a scope-b",
		ExpiresAt:    time.Now().Add(time.Hour),
		ClientID:     "client-id",
	}
	if err := SaveOAuthToken(path, token); err != nil {
		t.Fatalf("save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %04o", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if runtime.GOOS != "windows" && dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode = %04o", dirInfo.Mode().Perm())
	}
	status, err := OAuthStatus(path)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !status.Exists || !status.HasRefresh || len(status.Scopes) != 2 {
		t.Fatalf("unexpected status: %+v", status)
	}
	if err := ClearOAuthToken(path); err != nil {
		t.Fatalf("clear: %v", err)
	}
	status, err = OAuthStatus(path)
	if err != nil || status.Exists {
		t.Fatalf("status after clear: %+v, %v", status, err)
	}
}

func TestLoadOAuthTokenRejectsLoosePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions are not available")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}
	path := filepath.Join(dir, "token.json")
	if err := os.WriteFile(path, []byte(`{"refresh_token":"refresh"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadOAuthToken(path); err == nil || !strings.Contains(err.Error(), "require 0600") {
		t.Fatalf("expected permission error, got %v", err)
	}
}

func TestOAuthTokenEndpointErrorIsBounded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"expired code"}`))
	}))
	defer server.Close()
	provider, err := NewOAuthTokenProvider(OAuthOptions{
		ClientID:    "client-id",
		RedirectURI: "http://127.0.0.1:8888/callback",
		CachePath:   filepath.Join(t.TempDir(), "token.json"),
		AccountsURL: server.URL,
	})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err := provider.ExchangeCode(context.Background(), "code", "verifier"); err == nil || !strings.Contains(err.Error(), "expired code") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOAuthTokenEndpointTransientErrorsRemainAPIError(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if status == http.StatusTooManyRequests {
					w.Header().Set("Retry-After", "42")
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":"temporarily_unavailable","error_description":"try later"}`))
			}))
			defer server.Close()
			provider, err := NewOAuthTokenProvider(OAuthOptions{
				ClientID:    "client-id",
				RedirectURI: "http://127.0.0.1:8888/callback",
				CachePath:   filepath.Join(t.TempDir(), "token.json"),
				AccountsURL: server.URL,
			})
			if err != nil {
				t.Fatalf("provider: %v", err)
			}
			_, err = provider.ExchangeCode(context.Background(), "code", "verifier")
			var apiErr APIError
			if !errors.As(err, &apiErr) || errors.Is(err, ErrOAuthAuthentication) || apiErr.Status != status {
				t.Fatalf("unexpected error: %v", err)
			}
			if status == http.StatusTooManyRequests && apiErr.RetryAfter != 42*time.Second {
				t.Fatalf("retry after = %s", apiErr.RetryAfter)
			}
		})
	}
}

func TestGenerateOAuthState(t *testing.T) {
	first, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("generate state: %v", err)
	}
	second, err := GenerateOAuthState()
	if err != nil {
		t.Fatalf("generate second state: %v", err)
	}
	if first == "" || first == second || strings.ContainsAny(first+second, "+/=") {
		t.Fatalf("invalid OAuth states: %q %q", first, second)
	}
}

func TestNewOAuthTokenProviderValidation(t *testing.T) {
	if _, err := NewOAuthTokenProvider(OAuthOptions{CachePath: "token.json"}); !errors.Is(err, ErrOAuthAuthentication) {
		t.Fatalf("expected missing client ID error, got %v", err)
	}
	if _, err := NewOAuthTokenProvider(OAuthOptions{ClientID: "client-id"}); !errors.Is(err, ErrOAuthAuthentication) {
		t.Fatalf("expected missing cache path error, got %v", err)
	}
}

func TestOAuthAuthorizationURLValidation(t *testing.T) {
	provider, err := NewOAuthTokenProvider(OAuthOptions{
		ClientID:    "client-id",
		RedirectURI: "http://localhost:8888/callback",
		CachePath:   filepath.Join(t.TempDir(), "token.json"),
	})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err := provider.AuthorizationURL("", "challenge"); err == nil {
		t.Fatalf("expected missing state error")
	}
	if _, err := provider.AuthorizationURL("state", ""); err == nil {
		t.Fatalf("expected missing challenge error")
	}
	if _, err := provider.AuthorizationURL("state", "challenge"); err == nil {
		t.Fatalf("expected invalid redirect error")
	}
}

func TestOAuthExchangeValidationAndMalformedResponses(t *testing.T) {
	provider, err := NewOAuthTokenProvider(OAuthOptions{
		ClientID:    "client-id",
		RedirectURI: "http://127.0.0.1:8888/callback",
		CachePath:   filepath.Join(t.TempDir(), "token.json"),
	})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err := provider.ExchangeCode(context.Background(), "", "verifier"); err == nil {
		t.Fatalf("expected missing code error")
	}
	if _, err := provider.ExchangeCode(context.Background(), "code", ""); err == nil {
		t.Fatalf("expected missing verifier error")
	}

	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "missing fields", body: `{}`},
		{name: "missing refresh", body: `{"access_token":"access","expires_in":3600}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			p, err := NewOAuthTokenProvider(OAuthOptions{
				ClientID:    "client-id",
				RedirectURI: "http://127.0.0.1:8888/callback",
				CachePath:   filepath.Join(t.TempDir(), "token.json"),
				AccountsURL: server.URL,
			})
			if err != nil {
				t.Fatalf("provider: %v", err)
			}
			if _, err := p.ExchangeCode(context.Background(), "code", "verifier"); err == nil {
				t.Fatalf("expected malformed response error")
			}
		})
	}
}

func TestOAuthTokenCacheValidationErrors(t *testing.T) {
	if err := SaveOAuthToken("", OAuthToken{RefreshToken: "refresh"}); err == nil {
		t.Fatalf("expected empty path error")
	}
	if err := SaveOAuthToken(filepath.Join(t.TempDir(), "token.json"), OAuthToken{}); err == nil {
		t.Fatalf("expected missing refresh token error")
	}

	t.Run("not regular", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		if _, err := LoadOAuthToken(dir); err == nil {
			t.Fatalf("expected non-regular error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		path := filepath.Join(dir, "token.json")
		if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := LoadOAuthToken(path); err == nil {
			t.Fatalf("expected decode error")
		}
	})

	t.Run("no tokens", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		path := filepath.Join(dir, "token.json")
		if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := LoadOAuthToken(path); err == nil {
			t.Fatalf("expected empty token error")
		}
	})

	t.Run("loose directory", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("POSIX permissions are not available")
		}
		dir := t.TempDir()
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Fatalf("chmod: %v", err)
		}
		path := filepath.Join(dir, "token.json")
		if err := os.WriteFile(path, []byte(`{"refresh_token":"refresh"}`), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, err := LoadOAuthToken(path); err == nil || !strings.Contains(err.Error(), "require 0700") {
			t.Fatalf("expected directory permission error, got %v", err)
		}
	})

	t.Run("save parent is file", func(t *testing.T) {
		parent := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(parent, []byte("x"), 0o600); err != nil {
			t.Fatalf("write parent: %v", err)
		}
		if err := SaveOAuthToken(filepath.Join(parent, "token.json"), OAuthToken{RefreshToken: "refresh"}); err == nil {
			t.Fatalf("expected parent error")
		}
	})
}

func TestOAuthStatusAndClearErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := OAuthStatus(dir); err == nil {
		t.Fatalf("expected directory status error")
	}
	if err := ClearOAuthToken(""); err == nil {
		t.Fatalf("expected empty clear path error")
	}
	if err := ClearOAuthToken(filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatalf("clear missing: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write directory entry: %v", err)
	}
	if err := ClearOAuthToken(dir); err == nil {
		t.Fatalf("expected non-empty directory clear error")
	}
}

func TestOAuthProviderExpiredTokenWithoutRefresh(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "oauth", "default.json")
	if err := SaveOAuthToken(path, OAuthToken{
		AccessToken:  "expired",
		RefreshToken: "temporary",
		ExpiresAt:    time.Now().Add(-time.Hour),
		ClientID:     "client-id",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var token OAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		t.Fatalf("decode: %v", err)
	}
	token.RefreshToken = ""
	data, err = json.Marshal(token)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	provider, err := NewOAuthTokenProvider(OAuthOptions{ClientID: "client-id", CachePath: path})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err := provider.Token(context.Background()); !errors.Is(err, ErrOAuthAuthentication) || !strings.Contains(err.Error(), "no refresh token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOAuthTokenEndpointDoesNotFollowRedirects(t *testing.T) {
	var followed atomic.Bool
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		followed.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer sink.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, sink.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	provider, err := NewOAuthTokenProvider(OAuthOptions{
		ClientID:    "client-id",
		RedirectURI: "http://127.0.0.1:8888/callback",
		CachePath:   filepath.Join(t.TempDir(), "token.json"),
		AccountsURL: server.URL,
	})
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err := provider.ExchangeCode(context.Background(), "code", "verifier"); err == nil {
		t.Fatalf("expected redirect rejection")
	} else {
		var apiErr APIError
		if !errors.As(err, &apiErr) || apiErr.Status != http.StatusTemporaryRedirect {
			t.Fatalf("expected redirect API error, got %v", err)
		}
	}
	if followed.Load() {
		t.Fatalf("token endpoint redirect was followed")
	}
}

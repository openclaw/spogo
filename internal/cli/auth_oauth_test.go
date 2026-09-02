package cli

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/steipete/spogo/internal/config"
	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestAuthOAuthLoginCmd(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "test-code" {
			t.Errorf("unexpected token form: %v", r.Form)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "" {
			t.Errorf("PKCE exchange sent Authorization header")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access",
			"refresh_token": "refresh",
			"token_type":    "Bearer",
			"scope":         "user-library-read",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	redirectURI := "http://" + listener.Addr().String() + "/callback"
	_ = listener.Close()

	oldProvider := newOAuthTokenProvider
	newOAuthTokenProvider = func(opts spotify.OAuthOptions) (*spotify.OAuthTokenProvider, error) {
		opts.AccountsURL = tokenServer.URL
		return spotify.NewOAuthTokenProvider(opts)
	}
	t.Cleanup(func() { newOAuthTokenProvider = oldProvider })

	callbackErr := make(chan error, 1)
	oldOpen := openOAuthBrowser
	openOAuthBrowser = func(raw string) error {
		parsed, err := url.Parse(raw)
		if err != nil {
			return err
		}
		state := parsed.Query().Get("state")
		go func() {
			callbackURL := redirectURI + "?code=test-code&state=" + url.QueryEscape(state)
			var lastErr error
			for range 50 {
				resp, getErr := http.Get(callbackURL) //nolint:gosec // loopback test callback
				if getErr == nil {
					_ = resp.Body.Close()
					callbackErr <- nil
					return
				}
				lastErr = getErr
				time.Sleep(10 * time.Millisecond)
			}
			callbackErr <- lastErr
		}()
		return nil
	}
	t.Cleanup(func() { openOAuthBrowser = oldOpen })

	ctx, out, _ := testutil.NewTestContext(t, output.FormatJSON)
	ctx.Config = config.Default()
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	cmd := AuthOAuthLoginCmd{
		ClientID:    "client-id",
		RedirectURI: redirectURI,
		WaitTimeout: 2 * time.Second,
	}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := <-callbackErr; err != nil {
		t.Fatalf("callback: %v", err)
	}
	if ctx.Profile.Auth != "oauth" || ctx.Profile.SpotifyClientID != "client-id" {
		t.Fatalf("profile not updated: %+v", ctx.Profile)
	}
	if _, err := spotify.LoadOAuthToken(ctx.ResolveOAuthTokenPath()); err != nil {
		t.Fatalf("load cached token: %v", err)
	}
	var loginPayload map[string]any
	if err := json.Unmarshal(out.Bytes(), &loginPayload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if loginPayload["status"] != "ok" {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestAuthOAuthLoginAndClearSerializeLifecycle(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access",
			"refresh_token": "refresh",
			"token_type":    "Bearer",
			"scope":         "user-library-read",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	redirectURI := "http://" + listener.Addr().String() + "/callback"
	_ = listener.Close()

	oldProvider := newOAuthTokenProvider
	newOAuthTokenProvider = func(opts spotify.OAuthOptions) (*spotify.OAuthTokenProvider, error) {
		opts.AccountsURL = tokenServer.URL
		return spotify.NewOAuthTokenProvider(opts)
	}
	t.Cleanup(func() { newOAuthTokenProvider = oldProvider })

	oldOpen := openOAuthBrowser
	openOAuthBrowser = func(raw string) error {
		parsed, parseErr := url.Parse(raw)
		if parseErr != nil {
			return parseErr
		}
		state := parsed.Query().Get("state")
		go func() {
			callbackURL := redirectURI + "?code=***&state=" + url.QueryEscape(state)
			for range 50 {
				resp, getErr := http.Get(callbackURL) //nolint:gosec // loopback test callback
				if getErr == nil {
					_ = resp.Body.Close()
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
		return nil
	}
	t.Cleanup(func() { openOAuthBrowser = oldOpen })

	root := t.TempDir()
	configPath := filepath.Join(root, "config.toml")
	initialConfig := config.Default()
	initialConfig.SetProfile("default", config.Profile{Auth: "cookies"})
	if err := config.Save(configPath, initialConfig); err != nil {
		t.Fatalf("save initial config: %v", err)
	}

	loginConfig, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load login config: %v", err)
	}
	loginCtx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	loginCtx.Config = loginConfig
	loginCtx.ConfigPath = configPath
	loginCtx.ProfileKey = "default"
	loginCtx.Profile = loginConfig.Profile("default")

	// Load clear's context before login commits so it holds the stale cookie profile
	// that triggered the original token/profile race.
	clearConfig, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load clear config: %v", err)
	}
	clearCtx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	clearCtx.Config = clearConfig
	clearCtx.ConfigPath = configPath
	clearCtx.ProfileKey = "default"
	clearCtx.Profile = clearConfig.Profile("default")

	tokenSaved := make(chan struct{})
	releaseLogin := make(chan struct{})
	oldAfterExchange := afterOAuthTokenExchange
	afterOAuthTokenExchange = func() {
		close(tokenSaved)
		<-releaseLogin
	}
	t.Cleanup(func() {
		afterOAuthTokenExchange = oldAfterExchange
		select {
		case <-releaseLogin:
		default:
			close(releaseLogin)
		}
	})

	loginDone := make(chan error, 1)
	go func() {
		loginDone <- (&AuthOAuthLoginCmd{
			ClientID:    "client-id",
			RedirectURI: redirectURI,
			WaitTimeout: 2 * time.Second,
		}).Run(loginCtx)
	}()
	select {
	case <-tokenSaved:
	case <-time.After(2 * time.Second):
		t.Fatal("login did not reach the token/profile transition")
	}

	clearStarted := make(chan struct{})
	clearDone := make(chan error, 1)
	go func() {
		close(clearStarted)
		clearDone <- (&AuthOAuthClearCmd{}).Run(clearCtx)
	}()
	<-clearStarted
	select {
	case clearErr := <-clearDone:
		t.Fatalf("clear bypassed the login lifecycle lock: %v", clearErr)
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseLogin)

	select {
	case loginErr := <-loginDone:
		if loginErr != nil {
			t.Fatalf("login: %v", loginErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("login did not finish")
	}
	select {
	case clearErr := <-clearDone:
		if clearErr != nil {
			t.Fatalf("clear: %v", clearErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("clear did not finish")
	}

	finalConfig, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load final config: %v", err)
	}
	if auth := finalConfig.Profile("default").Auth; auth != "" {
		t.Fatalf("final auth = %q; want cookie fallback", auth)
	}
	if _, err := os.Stat(loginCtx.ResolveOAuthTokenPath()); !os.IsNotExist(err) {
		t.Fatalf("final token cache still exists: %v", err)
	}
}

func TestAuthOAuthStatusAndClearCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatJSON)
	ctx.Config = config.Default()
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	ctx.Profile = config.Profile{Auth: "oauth", SpotifyClientID: "client-id"}
	if err := spotify.SaveOAuthToken(ctx.ResolveOAuthTokenPath(), spotify.OAuthToken{
		AccessToken:  "access",
		RefreshToken: "refresh",
		Scope:        "scope-a scope-b",
		ExpiresAt:    time.Date(2026, 8, 27, 21, 0, 0, 0, time.UTC),
		ClientID:     "client-id",
	}); err != nil {
		t.Fatalf("save token: %v", err)
	}
	if err := (&AuthOAuthStatusCmd{}).Run(ctx); err != nil {
		t.Fatalf("status: %v", err)
	}
	var statusPayload map[string]any
	if err := json.Unmarshal(out.Bytes(), &statusPayload); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if statusPayload["authenticated"] != true {
		t.Fatalf("status omitted authentication: %s", out.String())
	}
	if _, ok := statusPayload["refresh_token"]; ok {
		t.Fatalf("status leaked refresh token: %s", out.String())
	}
	if _, ok := statusPayload["access_token"]; ok {
		t.Fatalf("status leaked access token: %s", out.String())
	}
	out.Reset()
	if err := (&AuthOAuthClearCmd{}).Run(ctx); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, err := os.Stat(ctx.ResolveOAuthTokenPath()); !os.IsNotExist(err) {
		t.Fatalf("expected token cache removed, got %v", err)
	}
	if ctx.Profile.Auth != "" {
		t.Fatalf("expected cookie auth restored, got %q", ctx.Profile.Auth)
	}
}

func TestAuthOAuthStatusRejectsClientIDMismatch(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	ctx.Profile = config.Profile{Auth: "oauth", SpotifyClientID: "configured-client"}
	if err := spotify.SaveOAuthToken(ctx.ResolveOAuthTokenPath(), spotify.OAuthToken{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
		ClientID:     "other-client",
	}); err != nil {
		t.Fatalf("save token: %v", err)
	}
	err := (&AuthOAuthStatusCmd{}).Run(ctx)
	if !errors.Is(err, spotify.ErrOAuthAuthentication) {
		t.Fatalf("expected OAuth authentication error, got %v", err)
	}
}

func TestAuthOAuthStatusRejectsInvalidCache(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	path := ctx.ResolveOAuthTokenPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
		t.Fatalf("write invalid cache: %v", err)
	}
	err := (&AuthOAuthStatusCmd{}).Run(ctx)
	if !errors.Is(err, spotify.ErrOAuthAuthentication) {
		t.Fatalf("expected OAuth authentication error, got %v", err)
	}
}

func TestAuthOAuthLoginRequiresClientID(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	err := (&AuthOAuthLoginCmd{RedirectURI: defaultOAuthRedirectURI}).Run(ctx)
	if !errors.Is(err, spotify.ErrOAuthAuthentication) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthOAuthStatusMissing(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	if err := (&AuthOAuthStatusCmd{}).Run(ctx); err != nil {
		t.Fatalf("status: %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "false\tcookies\t\tfalse" {
		t.Fatalf("status output = %q", got)
	}
}

func TestAuthOAuthClearMissingKeepsCookieSelection(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	ctx.Profile = config.Profile{Auth: "cookies"}
	ctx.Config = config.Default()
	ctx.Config.SetProfile("default", ctx.Profile)
	if err := config.Save(ctx.ConfigPath, ctx.Config); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := (&AuthOAuthClearCmd{}).Run(ctx); err != nil {
		t.Fatalf("clear missing: %v", err)
	}
	if ctx.Profile.Auth != "cookies" || strings.TrimSpace(out.String()) != "ok" {
		t.Fatalf("unexpected clear result: profile=%+v output=%q", ctx.Profile, out.String())
	}
}

func TestAuthOAuthClearKeepsTokenWhenProfileUpdateFails(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	root := t.TempDir()
	ctx.Config = config.Default()
	ctx.ConfigPath = filepath.Join(root, "config.toml")
	ctx.ProfileKey = "default"
	ctx.Profile = config.Profile{Auth: "oauth", SpotifyClientID: "client-id"}
	tokenPath := ctx.ResolveOAuthTokenPath()
	if err := spotify.SaveOAuthToken(tokenPath, spotify.OAuthToken{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresAt:    time.Now().Add(time.Hour),
		ClientID:     "client-id",
	}); err != nil {
		t.Fatalf("save token: %v", err)
	}
	if err := os.Mkdir(ctx.ConfigPath, 0o755); err != nil {
		t.Fatalf("block config save: %v", err)
	}

	if err := (&AuthOAuthClearCmd{}).Run(ctx); err == nil {
		t.Fatal("expected profile update failure")
	}
	if ctx.Profile.Auth != "oauth" {
		t.Fatalf("auth changed despite failed profile update: %q", ctx.Profile.Auth)
	}
	if _, err := spotify.LoadOAuthToken(tokenPath); err != nil {
		t.Fatalf("token removed before profile update succeeded: %v", err)
	}
}

func TestAuthOAuthLoginTimeoutAndBrowserError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	redirectURI := "http://" + listener.Addr().String() + "/callback"
	_ = listener.Close()

	oldOpen := openOAuthBrowser
	openOAuthBrowser = func(string) error { return errors.New("browser unavailable") }
	t.Cleanup(func() { openOAuthBrowser = oldOpen })

	ctx, _, errOut := testutil.NewTestContext(t, output.FormatPlain)
	ctx.Config = config.Default()
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	err = (&AuthOAuthLoginCmd{
		ClientID:    "client-id",
		RedirectURI: redirectURI,
		WaitTimeout: 20 * time.Millisecond,
	}).Run(ctx)
	if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("expected callback timeout, got %v", err)
	}
	if !strings.Contains(errOut.String(), "browser unavailable") {
		t.Fatalf("expected browser warning: %s", errOut.String())
	}
}

func TestAuthOAuthLoginNoOpenTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	redirectURI := "http://" + listener.Addr().String() + "/callback"
	_ = listener.Close()

	oldOpen := openOAuthBrowser
	openOAuthBrowser = func(string) error {
		t.Fatal("browser opener must not run with --no-open")
		return nil
	}
	t.Cleanup(func() { openOAuthBrowser = oldOpen })

	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	err = (&AuthOAuthLoginCmd{
		ClientID:    "client-id",
		RedirectURI: redirectURI,
		NoOpen:      true,
		WaitTimeout: 20 * time.Millisecond,
	}).Run(ctx)
	if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("expected callback timeout, got %v", err)
	}
}

func TestAuthOAuthLoginRejectsDeniedCallback(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	redirectURI := "http://" + listener.Addr().String() + "/callback"
	_ = listener.Close()

	oldOpen := openOAuthBrowser
	openOAuthBrowser = func(raw string) error {
		parsed, err := url.Parse(raw)
		if err != nil {
			return err
		}
		state := parsed.Query().Get("state")
		go func() {
			callbackURL := redirectURI + "?error=access_denied&state=" + url.QueryEscape(state)
			for range 50 {
				resp, getErr := http.Get(callbackURL) //nolint:gosec // loopback test callback
				if getErr == nil {
					_ = resp.Body.Close()
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
		return nil
	}
	t.Cleanup(func() { openOAuthBrowser = oldOpen })

	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	err = (&AuthOAuthLoginCmd{
		ClientID:    "client-id",
		RedirectURI: redirectURI,
		WaitTimeout: time.Second,
	}).Run(ctx)
	if err == nil || !strings.Contains(err.Error(), "access_denied") || !errors.Is(err, spotify.ErrOAuthAuthentication) {
		t.Fatalf("expected denial authentication error, got %v", err)
	}
}

func TestAuthOAuthLoginRedirectAndListenerValidation(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	if err := (&AuthOAuthLoginCmd{ClientID: "client-id", RedirectURI: "http://localhost:8888/callback"}).Run(ctx); err == nil {
		t.Fatalf("expected invalid redirect error")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = listener.Close() }()
	redirectURI := "http://" + listener.Addr().String() + "/callback"
	if err := (&AuthOAuthLoginCmd{ClientID: "client-id", RedirectURI: redirectURI}).Run(ctx); err == nil || !strings.Contains(err.Error(), "listen for spotify oauth callback") {
		t.Fatalf("expected listener error, got %v", err)
	}
}

func TestOpenBrowserURLLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux xdg-open command test")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "opened")
	script := filepath.Join(dir, "xdg-open")
	contents := "#!/bin/sh\nprintf '%s' \"$1\" > \"" + marker + "\"\n"
	if err := os.WriteFile(script, []byte(contents), 0o755); err != nil {
		t.Fatalf("write xdg-open: %v", err)
	}
	t.Setenv("PATH", dir)
	if err := openBrowserURL("https://example.test/authorize"); err != nil {
		t.Fatalf("open browser: %v", err)
	}
	for range 50 {
		data, err := os.ReadFile(marker)
		if err == nil {
			if string(data) != "https://example.test/authorize" {
				t.Fatalf("opened URL = %q", data)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("xdg-open helper did not run")
}

func TestAuthOAuthLoginRejectsMalformedCallback(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	redirectURI := "http://" + listener.Addr().String() + "/callback"
	_ = listener.Close()

	oldOpen := openOAuthBrowser
	openOAuthBrowser = func(raw string) error {
		parsed, err := url.Parse(raw)
		if err != nil {
			return err
		}
		state := parsed.Query().Get("state")
		go func() {
			callbackURL := redirectURI + "?state=" + url.QueryEscape(state)
			for range 50 {
				wrongStateURL := redirectURI + "?state=wrong"
				wrongResp, wrongErr := http.Get(wrongStateURL) //nolint:gosec // loopback test callback
				if wrongErr == nil {
					_ = wrongResp.Body.Close()
				}
				req, reqErr := http.NewRequest(http.MethodPost, callbackURL, nil)
				if reqErr != nil {
					return
				}
				resp, doErr := http.DefaultClient.Do(req) //nolint:gosec // loopback test callback
				if doErr == nil {
					_ = resp.Body.Close()
					resp, doErr = http.Get(callbackURL) //nolint:gosec // loopback test callback
					if doErr == nil {
						_ = resp.Body.Close()
					}
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
		return nil
	}
	t.Cleanup(func() { openOAuthBrowser = oldOpen })

	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.ConfigPath = filepath.Join(t.TempDir(), "config.toml")
	ctx.ProfileKey = "default"
	err = (&AuthOAuthLoginCmd{
		ClientID:    "client-id",
		RedirectURI: redirectURI,
		WaitTimeout: time.Second,
	}).Run(ctx)
	if err == nil || !strings.Contains(err.Error(), "missing the authorization code") {
		t.Fatalf("expected missing code error, got %v", err)
	}
}

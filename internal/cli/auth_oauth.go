package cli

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/steipete/spogo/internal/app"
	"github.com/steipete/spogo/internal/spotify"
)

type oauthCallback struct {
	code string
	err  error
}

type oauthStatusPayload struct {
	Authenticated bool     `json:"authenticated"`
	Auth          string   `json:"auth"`
	ClientID      string   `json:"client_id,omitempty"`
	ExpiresAt     string   `json:"expires_at,omitempty"`
	Expired       bool     `json:"expired"`
	HasRefresh    bool     `json:"has_refresh_token"`
	Scopes        []string `json:"scopes,omitempty"`
	TokenPath     string   `json:"token_path"`
	FileMode      string   `json:"file_mode,omitempty"`
}

const defaultOAuthRedirectURI = "http://127.0.0.1:8888/callback"

var (
	openOAuthBrowser      = openBrowserURL
	newOAuthTokenProvider = spotify.NewOAuthTokenProvider
)

func (cmd *AuthOAuthLoginCmd) Run(ctx *app.Context) error {
	clientID := firstNonEmpty(cmd.ClientID, ctx.Profile.SpotifyClientID)
	if clientID == "" {
		return fmt.Errorf("%w: pass --client-id, --spotify-client-id, or SPOGO_SPOTIFY_CLIENT_ID", spotify.ErrOAuthAuthentication)
	}
	redirectURI := firstNonEmpty(cmd.RedirectURI, ctx.Profile.SpotifyRedirectURI, defaultOAuthRedirectURI)
	if err := spotify.ValidateOAuthRedirectURI(redirectURI); err != nil {
		return err
	}
	provider, err := newOAuthTokenProvider(spotify.OAuthOptions{
		ClientID:    clientID,
		RedirectURI: redirectURI,
		CachePath:   ctx.ResolveOAuthTokenPath(),
		HTTPClient:  &http.Client{Timeout: ctx.EnsureTimeout()},
	})
	if err != nil {
		return err
	}
	verifier, challenge, err := spotify.GenerateOAuthPKCE()
	if err != nil {
		return err
	}
	state, err := spotify.GenerateOAuthState()
	if err != nil {
		return err
	}
	authorizationURL, err := provider.AuthorizationURL(state, challenge)
	if err != nil {
		return err
	}
	parsedRedirect, err := url.Parse(redirectURI)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", parsedRedirect.Host)
	if err != nil {
		return fmt.Errorf("listen for spotify oauth callback on %s: %w", parsedRedirect.Host, err)
	}
	defer func() { _ = listener.Close() }()

	callbackCh := make(chan oauthCallback, 1)
	mux := http.NewServeMux()
	callbackPath := parsedRedirect.Path
	if callbackPath == "" {
		callbackPath = "/"
	}
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Host != parsedRedirect.Host {
			http.Error(w, "invalid callback host", http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		query := r.URL.Query()
		if subtle.ConstantTimeCompare([]byte(query.Get("state")), []byte(state)) != 1 {
			http.Error(w, "invalid oauth state", http.StatusBadRequest)
			return
		}
		if oauthErr := query.Get("error"); oauthErr != "" {
			http.Error(w, "Spotify authorization was not granted. You can close this window.", http.StatusBadRequest)
			select {
			case callbackCh <- oauthCallback{err: fmt.Errorf("%w: spotify authorization failed: %s", spotify.ErrOAuthAuthentication, oauthErr)}:
			default:
			}
			return
		}
		code := query.Get("code")
		if code == "" {
			http.Error(w, "missing authorization code", http.StatusBadRequest)
			select {
			case callbackCh <- oauthCallback{err: fmt.Errorf("%w: callback is missing the authorization code", spotify.ErrOAuthAuthentication)}:
			default:
			}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, "<!doctype html><title>spogo authorized</title><p>Spotify authorization complete. You can close this window.</p>")
		select {
		case callbackCh <- oauthCallback{code: code}:
		default:
		}
	})
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErrCh := make(chan error, 1)
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			serveErrCh <- serveErr
		}
	}()
	defer func() { _ = server.Close() }()

	ctx.Output.Errorf("Spotify authorization URL: %s", authorizationURL)
	if !cmd.NoOpen {
		if err := openOAuthBrowser(authorizationURL); err != nil {
			ctx.Output.Errorf("Could not open a browser: %v", err)
		}
	}

	waitTimeout := cmd.WaitTimeout
	if waitTimeout <= 0 {
		waitTimeout = 5 * time.Minute
	}
	waitCtx, cancel := context.WithTimeout(ctx.CommandContext(), waitTimeout)
	defer cancel()
	var callback oauthCallback
	select {
	case callback = <-callbackCh:
	case serveErr := <-serveErrCh:
		return fmt.Errorf("spotify oauth callback server: %w", serveErr)
	case <-waitCtx.Done():
		return fmt.Errorf("spotify oauth callback wait failed: %w", waitCtx.Err())
	}
	if callback.err != nil {
		return callback.err
	}
	if _, err := provider.ExchangeCode(ctx.CommandContext(), callback.code, verifier); err != nil {
		return err
	}
	profile := ctx.Profile
	profile.Auth = "oauth"
	profile.SpotifyClientID = clientID
	profile.SpotifyRedirectURI = redirectURI
	if err := ctx.SaveProfile(profile); err != nil {
		return fmt.Errorf("oauth token saved but profile update failed: %w", err)
	}
	payload := map[string]any{
		"status":       "ok",
		"auth":         "oauth",
		"client_id":    clientID,
		"redirect_uri": redirectURI,
		"token_path":   ctx.ResolveOAuthTokenPath(),
	}
	return ctx.Output.Emit(payload, []string{"ok\toauth"}, []string{
		"Spotify OAuth login complete.",
		"Use --engine web for OAuth without browser cookies; Connect still requires cookies.",
	})
}

func (cmd *AuthOAuthStatusCmd) Run(ctx *app.Context) error {
	status, err := spotify.OAuthStatus(ctx.ResolveOAuthTokenPath())
	if err != nil {
		return fmt.Errorf("%w: invalid oauth token cache: %w", spotify.ErrOAuthAuthentication, err)
	}
	effectiveClientID := firstNonEmpty(ctx.Profile.SpotifyClientID, status.ClientID)
	if status.Exists && status.ClientID != "" && effectiveClientID != "" && status.ClientID != effectiveClientID {
		return fmt.Errorf("%w: cached token belongs to a different Spotify client ID", spotify.ErrOAuthAuthentication)
	}
	payload := oauthStatusPayload{
		Authenticated: status.Exists,
		Auth:          selectedAuth(ctx.Profile.Auth),
		ClientID:      effectiveClientID,
		Expired:       status.Expired,
		HasRefresh:    status.HasRefresh,
		Scopes:        status.Scopes,
		TokenPath:     ctx.ResolveOAuthTokenPath(),
	}
	if !status.ExpiresAt.IsZero() {
		payload.ExpiresAt = status.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if status.Exists {
		payload.FileMode = fmt.Sprintf("%04o", status.FileMode.Perm())
	}
	plain := []string{fmt.Sprintf("%t\t%s\t%s\t%t", payload.Authenticated, payload.Auth, payload.ExpiresAt, payload.HasRefresh)}
	human := []string{fmt.Sprintf("OAuth: %s", enabledLabel(payload.Authenticated))}
	if payload.Authenticated {
		human = append(
			human,
			fmt.Sprintf("Client ID: %s", payload.ClientID),
			fmt.Sprintf("Access token expires: %s (expired: %t)", payload.ExpiresAt, payload.Expired),
			fmt.Sprintf("Refresh token: %t", payload.HasRefresh),
			fmt.Sprintf("Token cache permissions: %s", payload.FileMode),
		)
	}
	return ctx.Output.Emit(payload, plain, human)
}

func (cmd *AuthOAuthClearCmd) Run(ctx *app.Context) error {
	path := ctx.ResolveOAuthTokenPath()
	if err := spotify.ClearOAuthToken(path); err != nil {
		return err
	}
	profile := ctx.Profile
	if selectedAuth(profile.Auth) == "oauth" {
		profile.Auth = ""
		if err := ctx.SaveProfile(profile); err != nil {
			return err
		}
	}
	payload := map[string]string{"status": "ok", "token_path": path}
	return ctx.Output.Emit(payload, []string{"ok"}, []string{"Cleared Spotify OAuth token cache."})
}

func selectedAuth(auth string) string {
	auth = strings.ToLower(strings.TrimSpace(auth))
	if auth == "" {
		return "cookies"
	}
	return auth
}

func enabledLabel(enabled bool) string {
	if enabled {
		return "authenticated"
	}
	return "not authenticated"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func openBrowserURL(rawURL string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
		args = []string{rawURL}
	case "windows":
		command = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", rawURL}
	default:
		command = "xdg-open"
		args = []string{rawURL}
	}
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

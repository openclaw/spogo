package spotify

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

const (
	defaultSpotifyAccountsURL = "https://accounts.spotify.com"
	oauthExpirySkew           = time.Minute
	oauthLockRetryDelay       = 25 * time.Millisecond
)

var ErrOAuthAuthentication = errors.New("spotify oauth authentication required")

var DefaultOAuthScopes = []string{
	"playlist-modify-private",
	"playlist-modify-public",
	"playlist-read-collaborative",
	"playlist-read-private",
	"user-follow-modify",
	"user-follow-read",
	"user-library-modify",
	"user-library-read",
	"user-modify-playback-state",
	"user-read-currently-playing",
	"user-read-playback-state",
	"user-read-private",
	"user-read-recently-played",
	"user-top-read",
}

type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
	ClientID     string    `json:"client_id"`
}

type OAuthTokenStatus struct {
	Exists     bool
	ClientID   string
	Scopes     []string
	ExpiresAt  time.Time
	Expired    bool
	HasRefresh bool
	FileMode   os.FileMode
}

type OAuthOptions struct {
	ClientID    string
	RedirectURI string
	Scopes      []string
	CachePath   string
	HTTPClient  *http.Client
	AccountsURL string
	Now         func() time.Time
}

type OAuthTokenProvider struct {
	opts OAuthOptions
	mu   sync.Mutex
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	Description  string `json:"error_description"`
}

func NewOAuthTokenProvider(opts OAuthOptions) (*OAuthTokenProvider, error) {
	opts.ClientID = strings.TrimSpace(opts.ClientID)
	if opts.ClientID == "" {
		return nil, fmt.Errorf("%w: spotify client ID is required", ErrOAuthAuthentication)
	}
	if opts.CachePath == "" {
		return nil, fmt.Errorf("%w: oauth token cache path is required", ErrOAuthAuthentication)
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: defaultHTTPClientTimeout}
	}
	httpClient := *opts.HTTPClient
	if httpClient.CheckRedirect == nil {
		httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	opts.HTTPClient = &httpClient
	if opts.AccountsURL == "" {
		opts.AccountsURL = defaultSpotifyAccountsURL
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if len(opts.Scopes) == 0 {
		opts.Scopes = append([]string(nil), DefaultOAuthScopes...)
	}
	return &OAuthTokenProvider{opts: opts}, nil
}

func (p *OAuthTokenProvider) Token(ctx context.Context) (Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	cacheLock, err := acquireOAuthCacheLock(ctx, p.opts.CachePath)
	if err != nil {
		return Token{}, err
	}
	defer releaseOAuthCacheLock(cacheLock)

	cached, err := loadOAuthTokenUnlocked(p.opts.CachePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Token{}, fmt.Errorf("%w: run 'spogo auth oauth login'", ErrOAuthAuthentication)
		}
		return Token{}, fmt.Errorf("%w: invalid oauth token cache: %w", ErrOAuthAuthentication, err)
	}
	if cached.ClientID != "" && cached.ClientID != p.opts.ClientID {
		return Token{}, fmt.Errorf("%w: cached token belongs to a different Spotify client ID", ErrOAuthAuthentication)
	}
	if cached.AccessToken != "" && cached.ExpiresAt.After(p.opts.Now().Add(oauthExpirySkew)) {
		return oauthAPIToken(cached), nil
	}
	if cached.RefreshToken == "" {
		return Token{}, fmt.Errorf("%w: cached token has no refresh token; run 'spogo auth oauth login'", ErrOAuthAuthentication)
	}
	refreshed, err := p.refresh(ctx, cached)
	if err != nil {
		return Token{}, err
	}
	if err := saveOAuthTokenUnlocked(p.opts.CachePath, refreshed); err != nil {
		return Token{}, err
	}
	return oauthAPIToken(refreshed), nil
}

func (p *OAuthTokenProvider) AuthorizationURL(state, codeChallenge string) (string, error) {
	if state == "" || codeChallenge == "" {
		return "", errors.New("oauth state and PKCE challenge are required")
	}
	if err := ValidateOAuthRedirectURI(p.opts.RedirectURI); err != nil {
		return "", err
	}
	params := url.Values{
		"client_id":             {p.opts.ClientID},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"redirect_uri":          {p.opts.RedirectURI},
		"response_type":         {"code"},
		"scope":                 {strings.Join(p.opts.Scopes, " ")},
		"state":                 {state},
	}
	return strings.TrimRight(p.opts.AccountsURL, "/") + "/authorize?" + params.Encode(), nil
}

func (p *OAuthTokenProvider) ExchangeCode(ctx context.Context, code, verifier string) (OAuthToken, error) {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(verifier) == "" {
		return OAuthToken{}, errors.New("authorization code and PKCE verifier are required")
	}
	form := url.Values{
		"client_id":     {p.opts.ClientID},
		"code":          {code},
		"code_verifier": {verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {p.opts.RedirectURI},
	}
	cacheLock, err := acquireOAuthCacheLock(ctx, p.opts.CachePath)
	if err != nil {
		return OAuthToken{}, err
	}
	defer releaseOAuthCacheLock(cacheLock)
	response, err := p.requestToken(ctx, form)
	if err != nil {
		return OAuthToken{}, err
	}
	token := p.tokenFromResponse(response)
	if token.RefreshToken == "" {
		return OAuthToken{}, errors.New("spotify oauth response did not include a refresh token")
	}
	if err := saveOAuthTokenUnlocked(p.opts.CachePath, token); err != nil {
		return OAuthToken{}, err
	}
	return token, nil
}

func (p *OAuthTokenProvider) refresh(ctx context.Context, previous OAuthToken) (OAuthToken, error) {
	form := url.Values{
		"client_id":     {p.opts.ClientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {previous.RefreshToken},
	}
	response, err := p.requestToken(ctx, form)
	if err != nil {
		return OAuthToken{}, fmt.Errorf("token refresh failed: %w", err)
	}
	token := p.tokenFromResponse(response)
	if token.RefreshToken == "" {
		token.RefreshToken = previous.RefreshToken
	}
	if response.Scope == "" {
		token.Scope = previous.Scope
	}
	if token.TokenType == "" {
		token.TokenType = previous.TokenType
	}
	return token, nil
}

func (p *OAuthTokenProvider) requestToken(ctx context.Context, form url.Values) (oauthTokenResponse, error) {
	endpoint := strings.TrimRight(p.opts.AccountsURL, "/") + "/api/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return oauthTokenResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.opts.HTTPClient.Do(req)
	if err != nil {
		return oauthTokenResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return oauthTokenResponse{}, err
	}
	var payload oauthTokenResponse
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			return oauthTokenResponse{}, fmt.Errorf("decode spotify oauth response: %w", err)
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := payload.Description
		if message == "" {
			message = payload.Error
		}
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return oauthTokenResponse{}, fmt.Errorf("%w: spotify oauth error (%d): %s", ErrOAuthAuthentication, resp.StatusCode, message)
		}
		return oauthTokenResponse{}, APIError{
			Status:     resp.StatusCode,
			Message:    message,
			Body:       string(body),
			RetryAfter: retryAfterFromResponse(resp),
		}
	}
	if payload.AccessToken == "" || payload.ExpiresIn <= 0 {
		return oauthTokenResponse{}, errors.New("spotify oauth response is missing access_token or expires_in")
	}
	return payload, nil
}

func (p *OAuthTokenProvider) tokenFromResponse(response oauthTokenResponse) OAuthToken {
	scope := response.Scope
	if scope == "" {
		scope = strings.Join(p.opts.Scopes, " ")
	}
	return OAuthToken{
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
		TokenType:    response.TokenType,
		Scope:        scope,
		ExpiresAt:    p.opts.Now().Add(time.Duration(response.ExpiresIn) * time.Second),
		ClientID:     p.opts.ClientID,
	}
}

func oauthAPIToken(token OAuthToken) Token {
	return Token{AccessToken: token.AccessToken, ExpiresAt: token.ExpiresAt, ClientID: token.ClientID}
}

func GenerateOAuthPKCE() (verifier, challenge string, err error) {
	verifier, err = randomURLSafe(64)
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(digest[:]), nil
}

func GenerateOAuthState() (string, error) {
	return randomURLSafe(32)
}

func randomURLSafe(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func ValidateOAuthRedirectURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid spotify redirect URI: %w", err)
	}
	if parsed.Scheme != "http" {
		return errors.New("spotify CLI redirect URI must use http on a loopback IP")
	}
	host := parsed.Hostname()
	if host != "127.0.0.1" && host != "::1" {
		return errors.New("spotify CLI redirect URI must use 127.0.0.1 or [::1], not localhost or a non-loopback host")
	}
	port := parsed.Port()
	if port == "" {
		return errors.New("spotify CLI redirect URI must include a port")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return errors.New("spotify CLI redirect URI must use a port from 1 to 65535")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return errors.New("spotify CLI redirect URI cannot include userinfo, a query, or a fragment")
	}
	return nil
}

func LoadOAuthToken(path string) (OAuthToken, error) {
	if _, err := os.Stat(path); err != nil {
		return OAuthToken{}, err
	}
	cacheLock, err := acquireOAuthCacheLock(context.Background(), path)
	if err != nil {
		return OAuthToken{}, err
	}
	defer releaseOAuthCacheLock(cacheLock)
	return loadOAuthTokenUnlocked(path)
}

func loadOAuthTokenUnlocked(path string) (OAuthToken, error) {
	info, err := os.Stat(path)
	if err != nil {
		return OAuthToken{}, err
	}
	if runtime.GOOS != "windows" {
		dirInfo, err := os.Stat(filepath.Dir(path))
		if err != nil {
			return OAuthToken{}, err
		}
		if dirInfo.Mode().Perm()&0o077 != 0 {
			return OAuthToken{}, fmt.Errorf("oauth token cache directory permissions are %04o; require 0700", dirInfo.Mode().Perm())
		}
	}
	if !info.Mode().IsRegular() {
		return OAuthToken{}, errors.New("oauth token cache is not a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return OAuthToken{}, fmt.Errorf("oauth token cache permissions are %04o; require 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return OAuthToken{}, err
	}
	var token OAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return OAuthToken{}, fmt.Errorf("decode oauth token cache: %w", err)
	}
	if token.RefreshToken == "" && token.AccessToken == "" {
		return OAuthToken{}, errors.New("oauth token cache contains no tokens")
	}
	return token, nil
}

func SaveOAuthToken(path string, token OAuthToken) error {
	cacheLock, err := acquireOAuthCacheLock(context.Background(), path)
	if err != nil {
		return err
	}
	defer releaseOAuthCacheLock(cacheLock)
	return saveOAuthTokenUnlocked(path, token)
}

func saveOAuthTokenUnlocked(path string, token OAuthToken) error {
	if path == "" {
		return errors.New("oauth token cache path is required")
	}
	if token.RefreshToken == "" {
		return errors.New("refusing to cache oauth token without refresh token")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dir, 0o700); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, ".oauth-token-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := replaceOAuthTokenFile(tmpPath, path); err != nil {
		return err
	}
	committed = true
	return nil
}

func OAuthStatus(path string) (OAuthTokenStatus, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return OAuthTokenStatus{}, nil
		}
		return OAuthTokenStatus{}, err
	}
	cacheLock, err := acquireOAuthCacheLock(context.Background(), path)
	if err != nil {
		return OAuthTokenStatus{}, err
	}
	defer releaseOAuthCacheLock(cacheLock)
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return OAuthTokenStatus{}, nil
		}
		return OAuthTokenStatus{}, err
	}
	token, err := loadOAuthTokenUnlocked(path)
	if err != nil {
		return OAuthTokenStatus{}, err
	}
	scopes := strings.Fields(token.Scope)
	return OAuthTokenStatus{
		Exists:     true,
		ClientID:   token.ClientID,
		Scopes:     scopes,
		ExpiresAt:  token.ExpiresAt,
		Expired:    !token.ExpiresAt.IsZero() && !token.ExpiresAt.After(time.Now()),
		HasRefresh: token.RefreshToken != "",
		FileMode:   info.Mode().Perm(),
	}, nil
}

func ClearOAuthToken(path string) error {
	if path == "" {
		return errors.New("oauth token cache path is required")
	}
	cacheLock, err := acquireOAuthCacheLock(context.Background(), path)
	if err != nil {
		return err
	}
	defer releaseOAuthCacheLock(cacheLock)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func WithOAuthLifecycleLock(ctx context.Context, path string, fn func() error) error {
	if path == "" {
		return errors.New("oauth token cache path is required")
	}
	lifecycleLock, err := acquireOAuthLifecycleLock(ctx, path)
	if err != nil {
		return err
	}
	defer releaseOAuthCacheLock(lifecycleLock)
	return fn()
}

func acquireOAuthCacheLock(ctx context.Context, path string) (*flock.Flock, error) {
	if path == "" {
		return nil, errors.New("oauth token cache path is required")
	}
	return acquireOAuthFileLock(ctx, path+".lock", filepath.Dir(path), "oauth token cache")
}

func acquireOAuthLifecycleLock(ctx context.Context, path string) (*flock.Flock, error) {
	return acquireOAuthFileLock(ctx, path+".lifecycle.lock", filepath.Dir(path), "oauth lifecycle")
}

func acquireOAuthFileLock(ctx context.Context, lockPath, dir, label string) (*flock.Flock, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	cacheLock := flock.New(lockPath, flock.SetPermissions(0o600))
	locked, err := cacheLock.TryLockContext(ctx, oauthLockRetryDelay)
	if err != nil {
		_ = cacheLock.Close()
		return nil, fmt.Errorf("lock %s: %w", label, err)
	}
	if !locked {
		_ = cacheLock.Close()
		return nil, fmt.Errorf("lock %s: %w", label, ctx.Err())
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(cacheLock.Path(), 0o600); err != nil {
			releaseOAuthCacheLock(cacheLock)
			return nil, err
		}
	}
	return cacheLock, nil
}

func releaseOAuthCacheLock(cacheLock *flock.Flock) {
	if cacheLock == nil {
		return
	}
	_ = cacheLock.Unlock()
	_ = cacheLock.Close()
}

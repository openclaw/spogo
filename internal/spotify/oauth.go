package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultSpotifyAccountsURL = "https://accounts.spotify.com"
	oauthExpirySkew           = time.Minute
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

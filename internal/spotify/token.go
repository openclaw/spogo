package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/steipete/spogo/internal/cookies"
)

type Token struct {
	AccessToken string
	ExpiresAt   time.Time
	Anonymous   bool
	ClientID    string
}

type TokenProvider interface {
	Token(ctx context.Context) (Token, error)
}

type CookieTokenProvider struct {
	Source  cookies.Source
	BaseURL string
	Client  *http.Client
	Timeout time.Duration
}

type tokenResponse struct {
	AccessToken                    string `json:"accessToken"`
	ExpiresIn                      int    `json:"expiresIn"`
	AccessTokenExpirationTimestamp int64  `json:"accessTokenExpirationTimestampMs"`
	IsAnonymous                    bool   `json:"isAnonymous"`
	ClientID                       string `json:"clientId"`
}

func newCookieTokenHTTPClient(jar http.CookieJar, timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = defaultHTTPClientTimeout
	}
	return &http.Client{Jar: jar, Timeout: timeout}
}

func (p CookieTokenProvider) Token(ctx context.Context) (Token, error) {
	if p.Source == nil {
		return Token{}, errors.New("cookie source required")
	}
	cookiesList, err := p.Source.Cookies(ctx)
	if err != nil {
		return Token{}, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return Token{}, err
	}
	base := p.BaseURL
	if base == "" {
		base = "https://open.spotify.com/"
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	baseURL, _ := url.Parse(base)
	jar.SetCookies(baseURL, cookiesList)
	client := p.Client
	if client == nil {
		client = newCookieTokenHTTPClient(jar, p.Timeout)
	} else {
		client.Jar = jar
	}
	code, version, err := generateTOTP(ctx, time.Now())
	if err != nil {
		return Token{}, err
	}
	params := url.Values{}
	params.Set("reason", "init")
	params.Set("productType", "web-player")
	params.Set("totp", code)
	params.Set("totpVer", strconv.Itoa(version))
	params.Set("totpServer", code)
	reqURL := base + "api/token?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return Token{}, err
	}
	applyRequestHeaders(req, requestHeaders{
		Accept:      "application/json",
		Language:    "en-US,en;q=0.9",
		AppPlatform: defaultSpotifyAppPlatform,
	})
	req.Header.Set("Origin", "https://open.spotify.com")
	req.Header.Set("Referer", "https://open.spotify.com/")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-CH-UA", "\"Chromium\";v=\"131\", \"Not_A Brand\";v=\"24\", \"Google Chrome\";v=\"131\"")
	req.Header.Set("Sec-CH-UA-Platform", "\"macOS\"")
	req.Header.Set("Sec-CH-UA-Mobile", "?0")
	resp, err := client.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Token{}, apiErrorFromResponse(resp)
	}
	var payload tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Token{}, err
	}
	if payload.AccessToken == "" {
		return Token{}, errors.New("missing access token")
	}
	expiresAt := time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	if payload.AccessTokenExpirationTimestamp > 0 {
		expiresAt = time.UnixMilli(payload.AccessTokenExpirationTimestamp)
	}
	return Token{
		AccessToken: payload.AccessToken,
		ExpiresAt:   expiresAt,
		Anonymous:   payload.IsAnonymous,
		ClientID:    payload.ClientID,
	}, nil
}

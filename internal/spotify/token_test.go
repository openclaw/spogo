package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubCookieSource struct{}

func (stubCookieSource) Cookies(ctx context.Context) ([]*http.Cookie, error) {
	return []*http.Cookie{{Name: "sp_dc", Value: "token", Domain: ".spotify.com", Path: "/"}}, nil
}

func TestCookieTokenProvider(t *testing.T) {
	restore := SetTotpSecretFetcher(func(ctx context.Context) (int, []byte, error) {
		return 1, []byte{1, 2, 3, 4}, nil
	})
	t.Cleanup(restore)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/token" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		query := r.URL.Query()
		if query.Get("reason") == "" || query.Get("productType") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if query.Get("totp") == "" || query.Get("totpVer") != "1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		payload := map[string]any{
			"accessToken":                      "abc",
			"accessTokenExpirationTimestampMs": time.Now().Add(1 * time.Hour).UnixMilli(),
			"isAnonymous":                      false,
		}
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()
	provider := CookieTokenProvider{Source: stubCookieSource{}, BaseURL: srv.URL + "/"}
	ok, err := provider.Token(context.Background())
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if ok.AccessToken != "abc" {
		t.Fatalf("token mismatch")
	}
	if time.Until(ok.ExpiresAt) <= 0 {
		t.Fatalf("expected expiry")
	}
}

func TestCookieTokenProviderMissingSource(t *testing.T) {
	provider := CookieTokenProvider{}
	if _, err := provider.Token(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCookieTokenProviderBadStatus(t *testing.T) {
	restore := SetTotpSecretFetcher(func(ctx context.Context) (int, []byte, error) {
		return 1, []byte{1, 2, 3, 4}, nil
	})
	t.Cleanup(restore)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	provider := CookieTokenProvider{Source: stubCookieSource{}, BaseURL: srv.URL + "/"}
	if _, err := provider.Token(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCookieTokenProviderMissingToken(t *testing.T) {
	restore := SetTotpSecretFetcher(func(ctx context.Context) (int, []byte, error) {
		return 1, []byte{1, 2, 3, 4}, nil
	})
	t.Cleanup(restore)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"accessToken": "", "isAnonymous": false})
	}))
	defer srv.Close()
	provider := CookieTokenProvider{Source: stubCookieSource{}, BaseURL: srv.URL + "/"}
	if _, err := provider.Token(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCookieTokenProviderFallbackClientTimeout(t *testing.T) {
	client := newCookieTokenHTTPClient(nil)
	if client.Timeout == 0 {
		t.Fatal("fallback HTTP client must set Timeout so cookie token fetches cannot hang")
	}
	t.Logf("fallback client Timeout=%s", client.Timeout)
}

type countingProvider struct {
	calls int
}

func (p *countingProvider) Token(ctx context.Context) (Token, error) {
	p.calls++
	return Token{AccessToken: "tok", ExpiresAt: time.Now().Add(2 * time.Minute)}, nil
}

func TestClientTokenCache(t *testing.T) {
	provider := &countingProvider{}
	client, err := NewClient(Options{TokenProvider: provider})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if _, err := client.token(context.Background()); err != nil {
		t.Fatalf("token: %v", err)
	}
	if _, err := client.token(context.Background()); err != nil {
		t.Fatalf("token: %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("expected one call, got %d", provider.calls)
	}
}

type errorProvider struct{}

func (errorProvider) Token(ctx context.Context) (Token, error) {
	return Token{}, errors.New("boom")
}

func TestClientTokenError(t *testing.T) {
	client, err := NewClient(Options{TokenProvider: errorProvider{}})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if _, err := client.token(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
}

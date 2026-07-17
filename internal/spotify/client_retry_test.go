package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type countingTokenProvider struct {
	calls int
}

func (p *countingTokenProvider) Token(context.Context) (Token, error) {
	p.calls++
	return Token{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func TestClientRetriesOnRateLimit(t *testing.T) {
	provider := &countingTokenProvider{}
	requests := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/me/player/devices", func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"status":  http.StatusTooManyRequests,
					"message": "rate limit",
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(deviceResponse{Devices: []deviceItem{{ID: "d1", Name: "Desk"}}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Options{
		TokenProvider: provider,
		BaseURL:       srv.URL,
		HTTPClient:    srv.Client(),
	})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	if _, err := client.Devices(context.Background()); err != nil {
		t.Fatalf("devices: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}
	if provider.calls < 2 {
		t.Fatalf("expected token refresh, got %d calls", provider.calls)
	}
}

// TestClientRetriesRateLimitedMutationAndSurfacesRetryAfter proves the default
// retry policy is unchanged: a mutating request (PUT /me/player/play) is still
// retried across all attempts on repeated 429s, and the final surfaced error
// carries the Retry-After from the last response so callers can see the
// cooldown hint. The hint is Spotify's earliest-retry guidance, not a
// guarantee that waiting it out clears the rate limit.
func TestClientRetriesRateLimitedMutationAndSurfacesRetryAfter(t *testing.T) {
	provider := &countingTokenProvider{}
	requests := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/me/player/play", func(w http.ResponseWriter, r *http.Request) {
		requests++
		// Non-final attempts advertise a short cooldown to keep the retry
		// delay small; the final attempt advertises the real cooldown that
		// must be surfaced to the caller.
		if requests < 3 {
			w.Header().Set("Retry-After", "1")
		} else {
			w.Header().Set("Retry-After", "42")
		}
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"status":  http.StatusTooManyRequests,
				"message": "rate limit",
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Options{
		TokenProvider: provider,
		BaseURL:       srv.URL,
		HTTPClient:    srv.Client(),
	})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	err = client.Play(context.Background(), "spotify:track:t1")
	if err == nil {
		t.Fatalf("expected rate limit error")
	}
	// Default policy retries all methods across the full 3 attempts.
	if requests != 3 {
		t.Fatalf("expected 3 requests, got %d", requests)
	}
	var apiErr APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Status != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", apiErr.Status)
	}
	if apiErr.RetryAfter != 42*time.Second {
		t.Fatalf("retry after = %v, want 42s", apiErr.RetryAfter)
	}
	if !strings.Contains(apiErr.Error(), "retry-after hint 42s") {
		t.Fatalf("expected retry-after hint in error string, got %q", apiErr.Error())
	}
}

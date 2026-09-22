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

func TestPlaybackResolvesConfiguredDeviceName(t *testing.T) {
	for _, test := range []struct {
		name, method, path, key, value string
		run                            func(*Client, context.Context) error
	}{
		{"play", http.MethodPut, "play", "", "", func(c *Client, ctx context.Context) error { return c.Play(ctx, "spotify:track:t1") }},
		{"pause", http.MethodPut, "pause", "", "", (*Client).Pause},
		{"next", http.MethodPost, "next", "", "", (*Client).Next},
		{"previous", http.MethodPost, "previous", "", "", (*Client).Previous},
		{"seek", http.MethodPut, "seek", "position_ms", "5000", func(c *Client, ctx context.Context) error { return c.Seek(ctx, 5000) }},
		{"volume", http.MethodPut, "volume", "volume_percent", "25", func(c *Client, ctx context.Context) error { return c.Volume(ctx, 25) }},
		{"shuffle", http.MethodPut, "shuffle", "state", "true", func(c *Client, ctx context.Context) error { return c.Shuffle(ctx, true) }},
		{"repeat", http.MethodPut, "repeat", "state", "track", func(c *Client, ctx context.Context) error { return c.Repeat(ctx, "track") }},
		{"queue", http.MethodPost, "queue", "uri", "spotify:track:t1", func(c *Client, ctx context.Context) error { return c.QueueAdd(ctx, "spotify:track:t1") }},
	} {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/me/player/devices", func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode(deviceResponse{Devices: []deviceItem{{ID: "device-id", Name: "Desk Speaker"}}})
			})
			calls := 0
			mux.HandleFunc("/me/player/"+test.path, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Query().Get("device_id") == "desk SPEAKER" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if r.Method != test.method || r.URL.Query().Get("device_id") != "device-id" {
					t.Errorf("unexpected playback request: %s %s", r.Method, r.URL)
				}
				if test.key != "" && r.URL.Query().Get(test.key) != test.value {
					t.Errorf("%s = %q, want %q", test.key, r.URL.Query().Get(test.key), test.value)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()
			client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL, Device: "desk SPEAKER"})
			if err != nil {
				t.Fatal(err)
			}
			if err := test.run(client, context.Background()); err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatalf("playback requests = %d, want rejected name then resolved ID", calls)
			}
		})
	}
}

func TestConfiguredDeviceDoesNotLeakToNonPlaybackMutation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/me/tracks", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("device_id"); got != "" {
			t.Errorf("unexpected device_id %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL, Device: "Desk Speaker"})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if err := client.LibraryModify(context.Background(), "/me/tracks", []string{"t1"}, http.MethodPut); err != nil {
		t.Fatalf("library modify: %v", err)
	}
}

func TestPlaybackPassesThroughDeviceIDWithoutLookup(t *testing.T) {
	const deviceID = "opaque-device:V2/one"
	deviceCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/me/player/devices", func(http.ResponseWriter, *http.Request) {
		deviceCalls++
	})
	mux.HandleFunc("/me/player/play", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("device_id"); got != deviceID {
			t.Errorf("device_id = %q, want %q", got, deviceID)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL, Device: deviceID})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if err := client.Play(context.Background(), "spotify:track:t1"); err != nil {
		t.Fatalf("play: %v", err)
	}
	if deviceCalls != 0 {
		t.Fatalf("device lookup calls = %d, want 0", deviceCalls)
	}
}

func TestPlaybackPreservesDeviceLookupAPIErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     int
		retryAfter string
	}{
		{name: "unauthorized", status: http.StatusUnauthorized},
		{name: "rate limited", status: http.StatusTooManyRequests, retryAfter: "42"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/me/player/devices", func(w http.ResponseWriter, _ *http.Request) {
				if test.retryAfter != "" {
					w.Header().Set("Retry-After", test.retryAfter)
				}
				w.WriteHeader(test.status)
			})
			playCalls := 0
			mux.HandleFunc("/me/player/play", func(w http.ResponseWriter, _ *http.Request) {
				playCalls++
				if playCalls > 1 {
					t.Error("play retried after lookup failure")
				}
				w.WriteHeader(http.StatusNotFound)
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL, Device: "Desk Speaker"})
			if err != nil {
				t.Fatalf("client: %v", err)
			}
			err = client.Play(context.Background(), "spotify:track:t1")
			var apiErr APIError
			if !errors.As(err, &apiErr) || apiErr.Status != test.status {
				t.Fatalf("error = %v, want API status %d", err, test.status)
			}
			if test.retryAfter != "" && apiErr.RetryAfter != 42*time.Second {
				t.Fatalf("retry after = %s, want 42s", apiErr.RetryAfter)
			}
		})
	}
}

func TestPlaybackPreservesCanceledDeviceLookup(t *testing.T) {
	client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: "http://127.0.0.1:1", Device: "Desk Speaker"})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Play(ctx, "spotify:track:t1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
}

func TestPlaybackRejectsNamedDeviceWithoutID(t *testing.T) {
	playCalls := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/me/player/devices", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"devices":[{"id":null,"name":"Desk Speaker"}]}`))
	})
	mux.HandleFunc("/me/player/play", func(w http.ResponseWriter, _ *http.Request) {
		playCalls++
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL, Device: "Desk Speaker"})
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	err = client.Play(context.Background(), "spotify:track:t1")
	if err == nil || err.Error() != `device "Desk Speaker" has no usable ID` {
		t.Fatalf("error = %v, want unusable device ID error", err)
	}
	if playCalls != 1 {
		t.Fatalf("playback requests = %d, want only rejected name", playCalls)
	}
}

func TestPlaybackPreservesMissingDeviceError(t *testing.T) {
	for _, devices := range []string{`[]`, `[{"id":"opaque-id","name":"Desk"}]`, `[{"id":"OPAQUE-ID","name":"Desk"}]`, `[{"id":"other-id","name":"opaque-id"},{"id":"opaque-id","name":"Desk"}]`} {
		t.Run(devices, func(t *testing.T) {
			playCalls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/me/player/devices" {
					_, _ = w.Write([]byte(`{"devices":` + devices + `}`))
					return
				}
				playCalls++
				w.WriteHeader(http.StatusNotFound)
			}))
			defer srv.Close()
			client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL, Device: "opaque-id"})
			if err != nil {
				t.Fatal(err)
			}
			var apiErr APIError
			if err := client.Play(context.Background(), ""); !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
				t.Fatalf("error = %v, want original 404", err)
			}
			if playCalls != 1 {
				t.Fatalf("playback calls = %d, want 1", playCalls)
			}
		})
	}
}

func TestPlaybackDoesNotResolveNamesAfterOtherErrors(t *testing.T) {
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/me/player/play" {
					t.Error("unexpected device lookup")
				}
				w.Header().Set("Retry-After", "42")
				w.WriteHeader(code)
			}))
			defer srv.Close()
			client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL, Device: "Desk Speaker"})
			if err != nil {
				t.Fatal(err)
			}
			var apiErr APIError
			if err := client.Play(context.Background(), ""); !errors.As(err, &apiErr) || apiErr.Status != code {
				t.Fatalf("error = %v, want %d", err, code)
			}
		})
	}
}

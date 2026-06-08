package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestConnectClientCommandRouteUsesMemoryAndCache(t *testing.T) {
	cache := newConnectCacheStore(filepath.Join(t.TempDir(), "connect.json"))
	client := &ConnectClient{
		cache:   cache,
		session: &connectSession{connectDeviceID: "connect-device"},
	}

	client.cacheCommandRoute(connectState{activeDeviceID: "active-device", originDeviceID: "origin-device"})

	from, to, ok := client.commandRoute()
	if !ok || from != "origin-device" || to != "active-device" {
		t.Fatalf("memory route = %q %q %v", from, to, ok)
	}

	restored := &ConnectClient{
		cache:   cache,
		session: &connectSession{},
	}
	from, to, ok = restored.commandRoute()
	if !ok || from != "origin-device" || to != "active-device" {
		t.Fatalf("cached route = %q %q %v", from, to, ok)
	}
	if restored.session.connectDeviceID != "connect-device" {
		t.Fatalf("expected cached connect device id, got %q", restored.session.connectDeviceID)
	}
}

func TestConnectClientCommandRouteFallsBackToConnectDevice(t *testing.T) {
	client := &ConnectClient{session: &connectSession{connectDeviceID: "connect-device"}}
	client.cacheCommandRoute(connectState{activeDeviceID: "active-device"})

	from, to, ok := client.commandRoute()
	if !ok || from != "connect-device" || to != "active-device" {
		t.Fatalf("route = %q %q %v", from, to, ok)
	}
}

func TestConnectClientCommandRouteExpiresAndInvalidates(t *testing.T) {
	cache := newConnectCacheStore(filepath.Join(t.TempDir(), "connect.json"))
	client := &ConnectClient{
		cache:                cache,
		session:              &connectSession{},
		cachedActiveDeviceID: "active-device",
		cachedOriginDeviceID: "origin-device",
		cachedRouteAt:        time.Now().Add(-commandRouteTTL - time.Second),
	}

	if _, _, ok := client.commandRoute(); ok {
		t.Fatal("expected expired memory route to miss")
	}

	client.cacheCommandRoute(connectState{activeDeviceID: "active-device", originDeviceID: "origin-device"})
	client.invalidateCommandRoute()
	if _, _, ok := client.commandRoute(); ok {
		t.Fatal("expected invalidated route to miss")
	}
}

func TestConnectClientArtistTopTracksUsesWebClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/artists/artist-id/top-tracks" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("market"); got != "NL" {
			t.Fatalf("market = %q, want NL", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tracks": []map[string]any{{
				"id":          "track-1",
				"uri":         "spotify:track:track-1",
				"name":        "Track One",
				"duration_ms": 123000,
				"album":       map[string]any{"name": "Album"},
				"artists":     []map[string]any{{"name": "Artist"}},
			}},
		})
	}))
	defer server.Close()

	web, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: server.URL, Market: "NL"})
	if err != nil {
		t.Fatalf("new web client: %v", err)
	}
	client := &ConnectClient{web: web}

	items, err := client.ArtistTopTracks(context.Background(), "artist-id", 10)
	if err != nil {
		t.Fatalf("artist top tracks: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Track One" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestConnectTopTimeRange(t *testing.T) {
	tests := map[string]string{
		"long_term":   "LONG_TERM",
		"medium_term": "MID_TERM",
		"short_term":  "SHORT_TERM",
		"custom":      "custom",
	}
	for input, want := range tests {
		if got := connectTopTimeRange(input); got != want {
			t.Fatalf("connectTopTimeRange(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsRouteStaleError(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusGone} {
		if !isRouteStaleError(APIError{Status: status}) {
			t.Fatalf("expected stale route for status %d", status)
		}
	}
	if isRouteStaleError(APIError{Status: http.StatusInternalServerError}) {
		t.Fatal("did not expect stale route for 500")
	}
	if isRouteStaleError(errors.New("plain error")) {
		t.Fatal("did not expect stale route for non-api error")
	}
}

func TestExtractRecentlyPlayedTracks(t *testing.T) {
	payload := map[string]any{
		"data": map[string]any{
			"lookup": []any{
				map[string]any{
					"_uri": "spotify:track:alias",
					"data": map[string]any{
						"uri":  "spotify:track:track-1",
						"name": "Track One",
						"artists": []any{
							map[string]any{"name": "Artist"},
						},
					},
				},
				"bad-item",
				map[string]any{"data": map[string]any{}},
			},
		},
	}

	tracks := extractRecentlyPlayedTracks(payload)
	if tracks["spotify:track:track-1"].Name != "Track One" {
		t.Fatalf("missing canonical track: %#v", tracks)
	}
	if tracks["spotify:track:alias"].Name != "Track One" {
		t.Fatalf("missing alias track: %#v", tracks)
	}
}

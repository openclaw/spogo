package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestConnectPlaybackCommands(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{
				"name":        "Desk",
				"device_type": "computer",
				"volume":      10,
			},
		},
		"player_state": map[string]any{
			"is_paused":   false,
			"position_ms": 100,
			"track": map[string]any{
				"uri":  "spotify:track:abc",
				"name": "Song",
			},
		},
		"active_device_id": "device-1",
	}
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case strings.Contains(req.URL.Path, "/connect/volume/"):
			if req.Method != http.MethodPut {
				return textResponse(http.StatusMethodNotAllowed, "method not allowed"), nil
			}
			return textResponse(http.StatusOK, "ok"), nil
		case req.Method == http.MethodPost:
			return textResponse(http.StatusOK, "ok"), nil
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newConnectClientForTests(transport)
	client.session.connectDeviceID = "device"
	client.session.connectionID = "conn"
	client.session.registeredAt = time.Now()

	if _, err := client.Playback(context.Background()); err != nil {
		t.Fatalf("playback: %v", err)
	}
	if _, err := client.Devices(context.Background()); err != nil {
		t.Fatalf("devices: %v", err)
	}
	if err := client.Play(context.Background(), ""); err != nil {
		t.Fatalf("play resume: %v", err)
	}
	if err := client.Play(context.Background(), "spotify:track:abc"); err != nil {
		t.Fatalf("play uri: %v", err)
	}
	if err := client.Play(context.Background(), "spotify:playlist:abc"); err != nil {
		t.Fatalf("play playlist: %v", err)
	}
	if err := client.Play(context.Background(), "spotify:album:xyz"); err != nil {
		t.Fatalf("play album: %v", err)
	}
	if err := client.Pause(context.Background()); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if err := client.Next(context.Background()); err != nil {
		t.Fatalf("next: %v", err)
	}
	if err := client.Previous(context.Background()); err != nil {
		t.Fatalf("previous: %v", err)
	}
	if err := client.Seek(context.Background(), -1); err != nil {
		t.Fatalf("seek: %v", err)
	}
	if err := client.Volume(context.Background(), -5); err != nil {
		t.Fatalf("volume: %v", err)
	}
	if err := client.Volume(context.Background(), 200); err != nil {
		t.Fatalf("volume high: %v", err)
	}
	if err := client.Shuffle(context.Background(), true); err != nil {
		t.Fatalf("shuffle: %v", err)
	}
	if err := client.Repeat(context.Background(), "track"); err != nil {
		t.Fatalf("repeat track: %v", err)
	}
	if err := client.Repeat(context.Background(), "context"); err != nil {
		t.Fatalf("repeat context: %v", err)
	}
	if err := client.Repeat(context.Background(), "off"); err != nil {
		t.Fatalf("repeat off: %v", err)
	}
	if err := client.QueueAdd(context.Background(), "spotify:track:abc"); err != nil {
		t.Fatalf("queue add: %v", err)
	}
	if _, err := client.Queue(context.Background()); err != nil {
		t.Fatalf("queue: %v", err)
	}
	if err := client.Transfer(context.Background(), "device-1"); err != nil {
		t.Fatalf("transfer: %v", err)
	}
}

func TestConnectPlaybackCachesDirectRoute(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{"name": "Desk", "device_type": "computer"},
		},
		"active_device_id": "device-1",
	}
	stateCalls := 0
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			stateCalls++
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/player/command/from/device/to/device-1"):
			return textResponse(http.StatusOK, "ok"), nil
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newRegisteredConnectClientForTests(transport)

	if _, err := client.Playback(context.Background()); err != nil {
		t.Fatalf("playback: %v", err)
	}
	if err := client.Pause(context.Background()); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if stateCalls != 1 {
		t.Fatalf("expected one state call, got %d", stateCalls)
	}
}

func TestConnectPlaybackPersistsDirectRoute(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{"name": "Desk", "device_type": "computer"},
		},
		"active_device_id": "device-1",
	}
	first := newRegisteredConnectClientForTests(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		default:
			return textResponse(http.StatusOK, "ok"), nil
		}
	}))
	attachConnectCacheForTests(first, cachePath)
	if _, err := first.Playback(context.Background()); err != nil {
		t.Fatalf("playback: %v", err)
	}

	second := newConnectClientForTests(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			t.Fatalf("unexpected state refresh")
			return textResponse(http.StatusInternalServerError, "unexpected state refresh"), nil
		case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/player/command/from/device/to/device-1"):
			return textResponse(http.StatusOK, "ok"), nil
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	}))
	attachConnectCacheForTests(second, cachePath)
	if err := second.Pause(context.Background()); err != nil {
		t.Fatalf("pause: %v", err)
	}
}

func TestConnectPlaybackStaleRouteFallsBackToState(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"fresh-device": map[string]any{"name": "Desk", "device_type": "computer"},
		},
		"player_state": map[string]any{
			"play_origin": map[string]any{"device_identifier": "origin-device"},
		},
		"active_device_id": "fresh-device",
	}
	sawStaleRoute := false
	sawFreshRoute := false
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/player/command/from/stale-origin/to/stale-device"):
			sawStaleRoute = true
			return textResponse(http.StatusGone, "gone"), nil
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/player/command/from/origin-device/to/fresh-device"):
			sawFreshRoute = true
			return textResponse(http.StatusOK, "ok"), nil
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newRegisteredConnectClientForTests(transport)
	client.cachedActiveDeviceID = "stale-device"
	client.cachedOriginDeviceID = "stale-origin"
	client.cachedRouteAt = time.Now()

	if err := client.Pause(context.Background()); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if !sawStaleRoute || !sawFreshRoute {
		t.Fatalf("expected stale and fresh routes, stale=%t fresh=%t", sawStaleRoute, sawFreshRoute)
	}
}

func TestConnectPlaybackHydratesSparseTrack(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{"name": "Desk", "device_type": "computer"},
		},
		"player_state": map[string]any{
			"is_paused": true,
			"track": map[string]any{
				"uri":  "spotify:track:t1",
				"name": "Song",
			},
		},
		"active_device_id": "device-1",
	}
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.URL.Query().Get("operationName") == "getTrack":
			return jsonResponse(http.StatusOK, map[string]any{
				"data": map[string]any{"track": map[string]any{
					"uri":     "spotify:track:t1",
					"name":    "Song",
					"artists": []any{map[string]any{"name": "Artist"}},
					"album":   map[string]any{"name": "Album"},
				}},
			}), nil
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newRegisteredConnectClientForTests(transport)
	client.hashes.hashes["getTrack"] = "hash"
	status, err := client.Playback(context.Background())
	if err != nil {
		t.Fatalf("playback: %v", err)
	}
	if status.Item == nil || len(status.Item.Artists) != 1 || status.Item.Artists[0] != "Artist" || status.Item.Album != "Album" {
		t.Fatalf("expected hydrated item: %#v", status.Item)
	}
}

func TestConnectPlaybackActiveDeviceFromDevices(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{
				"name":        "Desk",
				"device_type": "computer",
				"is_active":   true,
			},
		},
		"player_state": map[string]any{
			"is_paused": true,
		},
	}
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPost:
			return textResponse(http.StatusOK, "ok"), nil
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newConnectClientForTests(transport)
	client.session.connectDeviceID = "device"
	client.session.connectionID = "conn"
	client.session.registeredAt = time.Now()

	devices, err := client.Devices(context.Background())
	if err != nil {
		t.Fatalf("devices: %v", err)
	}
	if len(devices) != 1 || !devices[0].Active {
		t.Fatalf("expected active device: %#v", devices)
	}
	if err := client.Transfer(context.Background(), "device-1"); err != nil {
		t.Fatalf("transfer: %v", err)
	}
}

func TestConnectTransferFallsBackToWebAPIWithoutOriginDevice(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{
				"name":        "Desk",
				"device_type": "computer",
			},
		},
		"player_state": map[string]any{
			"is_paused": true,
		},
	}
	var sawWebTransfer bool
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPut && req.URL.Path == "/v1/me/player":
			sawWebTransfer = true
			return textResponse(http.StatusNoContent, ""), nil
		case req.Method == http.MethodPost:
			t.Fatalf("unexpected connect command: %s", req.URL.Path)
			return textResponse(http.StatusInternalServerError, "unexpected connect command"), nil
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newRegisteredConnectClientForTests(transport)
	webClient, err := NewClient(Options{
		TokenProvider: staticTokenProvider{},
		HTTPClient:    client.client,
	})
	if err != nil {
		t.Fatalf("new web client: %v", err)
	}
	client.web = webClient

	if err := client.Transfer(context.Background(), "device-1"); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if !sawWebTransfer {
		t.Fatalf("expected web transfer fallback")
	}
}

func TestConnectPlayFallsBackToWebAPIWithoutActiveDevice(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{
				"name":        "Desk",
				"device_type": "computer",
			},
		},
		"player_state": map[string]any{
			"is_paused": true,
		},
	}
	var sawWebPlay bool
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPut && req.URL.Path == "/v1/me/player/play":
			sawWebPlay = true
			return textResponse(http.StatusNoContent, ""), nil
		case req.Method == http.MethodPost:
			t.Fatalf("unexpected connect command: %s", req.URL.Path)
			return nil, errors.New("unexpected connect command")
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newRegisteredConnectClientForTests(transport)
	webClient, err := NewClient(Options{
		TokenProvider: staticTokenProvider{},
		HTTPClient:    client.client,
	})
	if err != nil {
		t.Fatalf("new web client: %v", err)
	}
	client.web = webClient

	if err := client.Play(context.Background(), "spotify:track:abc"); err != nil {
		t.Fatalf("play: %v", err)
	}
	if !sawWebPlay {
		t.Fatalf("expected web play fallback")
	}
}

func TestConnectPlayUsesConfiguredDeviceWithoutActiveDevice(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{
				"name":        "Desk",
				"device_type": "computer",
			},
		},
		"player_state": map[string]any{
			"is_paused": true,
		},
	}
	var sawConnectPlay bool
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/player/command/from/device/to/device-1"):
			sawConnectPlay = true
			return textResponse(http.StatusOK, "ok"), nil
		case req.Method == http.MethodPut && req.URL.Path == "/v1/me/player/play":
			t.Fatalf("unexpected web play fallback")
			return nil, errors.New("unexpected web play fallback")
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newRegisteredConnectClientForTests(transport)
	client.device = "Desk"

	if err := client.Play(context.Background(), "spotify:track:abc"); err != nil {
		t.Fatalf("play: %v", err)
	}
	if !sawConnectPlay {
		t.Fatalf("expected connect play")
	}
}

func TestSendPlayerCommandMissingDevice(t *testing.T) {
	client := newConnectClientForTests(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return textResponse(http.StatusOK, ""), nil
	}))
	err := client.sendPlayerCommand(context.Background(), connectState{}, "pause", nil)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRandomHexAndOrigin(t *testing.T) {
	if randomHex(0) != "" {
		t.Fatalf("expected empty")
	}
	value := randomHex(6)
	if len(value) != 6 {
		t.Fatalf("unexpected length: %s", value)
	}
	player := map[string]any{"play_origin": map[string]any{"device_identifier": "abc"}}
	if mapPlayOriginID(player) != "abc" {
		t.Fatalf("expected origin id")
	}
	if connectVersion(connectAuth{ClientVersion: "a"}) != "a" {
		t.Fatalf("expected client version fallback")
	}
	if connectVersion(connectAuth{ClientVersion: "a", ConnectVersion: "b"}) != "b" {
		t.Fatalf("expected connect version")
	}
}

func TestConnectPlaybackErrorPaths(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return textResponse(http.StatusInternalServerError, "fail"), nil
	})
	client := newConnectClientForTests(transport)
	client.session.connectDeviceID = "device"
	client.session.connectionID = "conn"
	client.session.registeredAt = time.Now()

	if _, err := client.Playback(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := client.Devices(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
	if err := client.Pause(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
	if err := client.Next(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
	if err := client.Previous(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
	if err := client.Shuffle(context.Background(), false); err == nil {
		t.Fatalf("expected error")
	}
	if err := client.QueueAdd(context.Background(), "spotify:track:abc"); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := client.Queue(context.Background()); err == nil {
		t.Fatalf("expected error")
	}
	if err := client.Transfer(context.Background(), "device-1"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestConnectPlayContextURIPayload(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{"name": "Desk", "device_type": "computer"},
		},
		"player_state": map[string]any{
			"is_paused":   false,
			"position_ms": 0,
		},
		"active_device_id": "device-1",
	}
	var capturedBody string
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPost:
			b, _ := io.ReadAll(req.Body)
			capturedBody = string(b)
			return textResponse(http.StatusOK, "ok"), nil
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})

	newClient := func() *ConnectClient {
		c := newConnectClientForTests(transport)
		c.session.connectDeviceID = "device"
		c.session.connectionID = "conn"
		c.session.registeredAt = time.Now()
		return c
	}

	// Context URI (playlist) — must use "context" field, not "track_uri"
	capturedBody = ""
	if err := newClient().Play(context.Background(), "spotify:playlist:pl1"); err != nil {
		t.Fatalf("play playlist: %v", err)
	}
	if !strings.Contains(capturedBody, `"context"`) {
		t.Errorf("playlist play: expected context field in body, got: %s", capturedBody)
	}
	if strings.Contains(capturedBody, `"track_uri"`) {
		t.Errorf("playlist play: unexpected track_uri in body: %s", capturedBody)
	}

	// Track URI — must use "track_uri" and also include "context" (track as its own context)
	capturedBody = ""
	if err := newClient().Play(context.Background(), "spotify:track:t1"); err != nil {
		t.Fatalf("play track: %v", err)
	}
	if !strings.Contains(capturedBody, `"track_uri"`) {
		t.Errorf("track play: expected track_uri field in body, got: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, `"context"`) {
		t.Errorf("track play: expected context field in body, got: %s", capturedBody)
	}
}

func TestSendConnectCommandHTTPError(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return textResponse(http.StatusInternalServerError, "fail"), nil
	})
	client := newConnectClientForTests(transport)
	client.session.token = Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour)}
	client.session.clientToken = "ct"
	client.session.clientTokenT = time.Now().Add(time.Hour)
	client.session.clientVer = "1.0.0"
	client.session.deviceID = "device"

	if err := client.sendConnectCommand(context.Background(), "https://example.com", map[string]any{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestConnectQueueHydratesSparseTracksWithBoundedConcurrency(t *testing.T) {
	const queuedCount = 12
	next := make([]any, 0, queuedCount)
	for index := 0; index < queuedCount; index++ {
		entry := map[string]any{"uri": fmt.Sprintf("spotify:track:t%d", index)}
		if index == 0 {
			entry["metadata"] = map[string]any{"title": "Ready", "artist_name": "Ready Artist", "album_title": "Ready Album"}
		}
		next = append(next, entry)
	}
	statePayload := map[string]any{
		"devices":          map[string]any{"device-1": map[string]any{"name": "Desk"}},
		"active_device_id": "device-1",
		"player_state": map[string]any{
			"track":       map[string]any{"uri": "spotify:track:current"},
			"next_tracks": next,
		},
	}
	var active, maximum, lookups atomic.Int32
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.URL.Host == "api-partner.spotify.com" && req.URL.Query().Get("operationName") == "getTrack":
			lookups.Add(1)
			current := active.Add(1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			active.Add(-1)
			var variables map[string]any
			if err := json.Unmarshal([]byte(req.URL.Query().Get("variables")), &variables); err != nil {
				t.Errorf("decode track variables: %v", err)
				return textResponse(http.StatusBadRequest, "invalid variables"), nil
			}
			uri := getString(variables, "uri")
			if uri == "spotify:track:t7" {
				return textResponse(http.StatusInternalServerError, "track unavailable"), nil
			}
			return jsonResponse(http.StatusOK, map[string]any{"data": map[string]any{"trackUnion": map[string]any{
				"uri":         uri,
				"name":        "Track " + idFromURI(uri),
				"artists":     []any{map[string]any{"name": "Artist " + idFromURI(uri)}},
				"album":       map[string]any{"name": "Album " + idFromURI(uri)},
				"duration_ms": 123456,
			}}}), nil
		default:
			t.Errorf("unexpected request outside internal GraphQL: %s", req.URL.Redacted())
			return textResponse(http.StatusNotFound, "unexpected"), nil
		}
	})
	client := newRegisteredConnectClientForTests(transport)
	client.hashes.hashes["getTrack"] = "hash"
	queue, err := client.Queue(context.Background())
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if queue.CurrentlyPlaying == nil || queue.CurrentlyPlaying.Name != "Track current" || queue.CurrentlyPlaying.Album != "Album current" {
		t.Fatalf("current track was not hydrated: %#v", queue.CurrentlyPlaying)
	}
	if len(queue.Queue) != queuedCount {
		t.Fatalf("queue length = %d, want %d", len(queue.Queue), queuedCount)
	}
	for index, item := range queue.Queue {
		switch index {
		case 0:
			if item.Name != "Ready" {
				t.Fatalf("complete metadata was replaced: %#v", item)
			}
		case 7:
			if item.Name != "" {
				t.Fatalf("failed lookup unexpectedly hydrated: %#v", item)
			}
		default:
			if item.Name == "" || len(item.Artists) == 0 || item.Album == "" || item.DurationMS != 123456 {
				t.Fatalf("queue entry %d remains incomplete: %#v", index, item)
			}
		}
	}
	if got := lookups.Load(); got != queuedCount {
		t.Fatalf("internal GraphQL lookups = %d, want %d", got, queuedCount)
	}
	if got := maximum.Load(); got <= 1 || got > queueHydrationConcurrency {
		t.Fatalf("maximum queue hydration concurrency = %d, want between 2 and %d", got, queueHydrationConcurrency)
	}
}

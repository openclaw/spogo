package spotify

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
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

func TestConnectPlayTracksFallsBackToWebAPI(t *testing.T) {
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
		"active_device_id": "device-1",
	}
	var sawWebPlay bool
	var webBody string
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPut && req.URL.Path == "/v1/me/player/play":
			sawWebPlay = true
			b, _ := io.ReadAll(req.Body)
			webBody = string(b)
			return textResponse(http.StatusNoContent, ""), nil
		case req.Method == http.MethodPost:
			t.Fatalf("unexpected connect command for multi-track: %s", req.URL.Path)
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

	if err := client.PlayTracks(context.Background(), []string{"spotify:track:t1", "spotify:track:t2"}); err != nil {
		t.Fatalf("play tracks: %v", err)
	}
	if !sawWebPlay {
		t.Fatalf("expected web play fallback for multi-track")
	}
	if !strings.Contains(webBody, `"uris"`) {
		t.Fatalf("expected uris array in web body, got: %s", webBody)
	}
	i1 := strings.Index(webBody, "spotify:track:t1")
	i2 := strings.Index(webBody, "spotify:track:t2")
	if i1 < 0 || i2 < 0 || i1 > i2 {
		t.Fatalf("expected t1 before t2 in ordered uris, got: %s", webBody)
	}
}

func TestConnectPlayTracksResolvesConfiguredDeviceNameForWebAPI(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{
				"name":        "Desk",
				"device_type": "computer",
			},
		},
		"active_device_id": "device-1",
	}
	var webDeviceID string
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPut && req.URL.Path == "/v1/me/player/play":
			webDeviceID = req.URL.Query().Get("device_id")
			return textResponse(http.StatusNoContent, ""), nil
		case req.Method == http.MethodPost:
			t.Fatalf("unexpected connect command for multi-track: %s", req.URL.Path)
			return nil, errors.New("unexpected connect command")
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newRegisteredConnectClientForTests(transport)
	client.device = "Desk"
	webClient, err := NewClient(Options{
		TokenProvider: staticTokenProvider{},
		HTTPClient:    client.client,
		Device:        client.device,
	})
	if err != nil {
		t.Fatalf("new web client: %v", err)
	}
	client.web = webClient

	if err := client.PlayTracks(context.Background(), []string{"spotify:track:t1", "spotify:track:t2"}); err != nil {
		t.Fatalf("play tracks: %v", err)
	}
	if webDeviceID != "device-1" {
		t.Fatalf("expected resolved device id device-1, got %q", webDeviceID)
	}
}

func TestConnectPlayTracksPreservesConfiguredDeviceIDForWebAPI(t *testing.T) {
	statePayload := map[string]any{
		"devices": map[string]any{
			"device-1": map[string]any{
				"name":        "Desk",
				"device_type": "computer",
			},
		},
		"active_device_id": "device-1",
	}
	var webDeviceID string
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPut && req.URL.Path == "/v1/me/player/play":
			webDeviceID = req.URL.Query().Get("device_id")
			return textResponse(http.StatusNoContent, ""), nil
		case req.Method == http.MethodPost:
			t.Fatalf("unexpected connect command for multi-track: %s", req.URL.Path)
			return nil, errors.New("unexpected connect command")
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newRegisteredConnectClientForTests(transport)
	client.device = "device-1"
	webClient, err := NewClient(Options{
		TokenProvider: staticTokenProvider{},
		HTTPClient:    client.client,
		Device:        client.device,
	})
	if err != nil {
		t.Fatalf("new web client: %v", err)
	}
	client.web = webClient

	if err := client.PlayTracks(context.Background(), []string{"spotify:track:t1", "spotify:track:t2"}); err != nil {
		t.Fatalf("play tracks: %v", err)
	}
	if webDeviceID != "device-1" {
		t.Fatalf("expected device id device-1, got %q", webDeviceID)
	}
}

func TestConnectPlayTracksSingleUsesConnect(t *testing.T) {
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
	var sawConnectPlay bool
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodPut && strings.Contains(req.URL.Path, "/devices/hobs_"):
			return jsonResponse(http.StatusOK, statePayload), nil
		case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/player/command/"):
			sawConnectPlay = true
			return textResponse(http.StatusOK, "ok"), nil
		case req.Method == http.MethodPut && req.URL.Path == "/v1/me/player/play":
			t.Fatalf("unexpected web fallback for single track")
			return nil, errors.New("unexpected web fallback")
		default:
			return textResponse(http.StatusNotFound, "missing"), nil
		}
	})
	client := newConnectClientForTests(transport)
	client.session.connectDeviceID = "device"
	client.session.connectionID = "conn"
	client.session.registeredAt = time.Now()

	if err := client.PlayTracks(context.Background(), []string{"spotify:track:t1"}); err != nil {
		t.Fatalf("play tracks single: %v", err)
	}
	if !sawConnectPlay {
		t.Fatalf("expected native connect play for single track")
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

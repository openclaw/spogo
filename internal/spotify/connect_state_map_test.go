package spotify

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMapPlaybackStatusAndDevices(t *testing.T) {
	state := connectState{
		activeDeviceID: "device-1",
		devices: map[string]any{
			"device-1": map[string]any{
				"name":           "Desk",
				"device_type":    "computer",
				"volume_percent": 40,
			},
			"device-2": map[string]any{
				"device_name": "Phone",
				"device_type": "smartphone",
				"volume":      32768,
			},
		},
		playerState: map[string]any{
			"is_paused":   true,
			"position_ms": 1200,
			"shuffle":     true,
			"repeat":      "context",
			"track": map[string]any{
				"uri":  "spotify:track:abc",
				"name": "Song",
			},
		},
	}
	status := mapPlaybackStatus(state)
	if status.IsPlaying {
		t.Fatalf("expected paused")
	}
	if status.Device.ID != "device-1" || status.Device.Name != "Desk" {
		t.Fatalf("unexpected device: %#v", status.Device)
	}
	devices := mapDevices(state)
	var phone Device
	for _, device := range devices {
		if device.ID == "device-2" {
			phone = device
			break
		}
	}
	if phone.Volume != 50 {
		t.Fatalf("expected normalized volume, got %d", phone.Volume)
	}
	if status.Item == nil || status.Item.URI != "spotify:track:abc" {
		t.Fatalf("expected item")
	}
}

func TestMapQueue(t *testing.T) {
	state := connectState{
		playerState: map[string]any{
			"track": map[string]any{
				"uri":  "spotify:track:now",
				"name": "Now",
			},
			"next_tracks": []any{
				map[string]any{"track": map[string]any{"uri": "spotify:track:next", "name": "Next"}},
			},
		},
	}
	queue := mapQueue(state)
	if queue.CurrentlyPlaying == nil || queue.CurrentlyPlaying.URI != "spotify:track:now" {
		t.Fatalf("expected current item")
	}
	if len(queue.Queue) != 1 || queue.Queue[0].URI != "spotify:track:next" {
		t.Fatalf("expected next item")
	}
}

func TestExtractPlaybackTrackContext(t *testing.T) {
	player := map[string]any{
		"context_uri": "spotify:album:abc",
	}
	item := extractPlaybackTrack(player)
	if item.URI != "spotify:album:abc" || item.Type != "album" {
		t.Fatalf("unexpected item: %#v", item)
	}
}

func TestExtractPlaybackTrackCurrent(t *testing.T) {
	player := map[string]any{
		"current_track": map[string]any{
			"uri":  "spotify:track:xyz",
			"name": "Song",
		},
	}
	item := extractPlaybackTrack(player)
	if item.URI != "spotify:track:xyz" {
		t.Fatalf("unexpected item: %#v", item)
	}
}

func TestConnectTrackMetadataMapping(t *testing.T) {
	tests := []struct {
		name     string
		entry    map[string]any
		wantName string
		artists  []string
		album    string
		duration int
		explicit bool
		known    bool
	}{
		{
			name: "ordered connect metadata",
			entry: map[string]any{
				"uri": "spotify:track:one",
				"metadata": map[string]any{
					"title":         "Sun & You",
					"artist_name:2": "Third Artist",
					"artist_name":   "Stoneface & Terminal",
					"artist_name:1": "Second Artist",
					"album_title":   "Sun & You",
					"duration":      "207884",
					"is_explicit":   "false",
				},
			},
			wantName: "Sun & You",
			artists:  []string{"Stoneface & Terminal", "Second Artist", "Third Artist"},
			album:    "Sun & You",
			duration: 207884,
			known:    true,
		},
		{
			name: "nested track metadata",
			entry: map[string]any{"track": map[string]any{
				"uri": "spotify:track:two",
				"metadata": map[string]any{
					"title":       "Nested",
					"artist_name": "Artist",
					"album_title": "Album",
					"is_explicit": "true",
				},
			}},
			wantName: "Nested",
			artists:  []string{"Artist"},
			album:    "Album",
			explicit: true,
			known:    true,
		},
		{
			name: "invalid metadata preserves existing values",
			entry: map[string]any{
				"uri":         "spotify:track:three",
				"name":        "Existing",
				"duration_ms": 123,
				"metadata": map[string]any{
					"artist_name:bad": "Ignored",
					"duration":        "invalid",
					"is_explicit":     "unknown",
				},
			},
			wantName: "Existing",
			duration: 123,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			item, ok := extractConnectTrack(test.entry)
			if !ok {
				t.Fatal("expected track")
			}
			artistsMatch := reflect.DeepEqual(item.Artists, test.artists) || len(item.Artists) == 0 && len(test.artists) == 0
			if item.Name != test.wantName || !artistsMatch || item.Album != test.album || item.DurationMS != test.duration || item.Explicit != test.explicit || item.ExplicitKnown != test.known {
				t.Fatalf("unexpected mapped metadata: %#v", item)
			}
		})
	}
}

func TestMapQueueUsesConnectMetadataForCurrentAndUpcoming(t *testing.T) {
	state := connectState{playerState: map[string]any{
		"track": map[string]any{
			"uri":      "spotify:track:current",
			"metadata": map[string]any{"title": "Current", "artist_name": "Current Artist", "album_title": "Current Album"},
		},
		"next_tracks": []any{map[string]any{
			"uri": "spotify:track:next",
			"metadata": map[string]any{
				"title": "Next", "artist_name": "Next Artist", "album_title": "Next Album", "duration": "1234", "is_explicit": "false",
			},
		}},
	}}
	queue := mapQueue(state)
	if queue.CurrentlyPlaying == nil || queue.CurrentlyPlaying.Album != "Current Album" {
		t.Fatalf("unexpected current track: %#v", queue.CurrentlyPlaying)
	}
	if len(queue.Queue) != 1 || queue.Queue[0].Name != "Next" || queue.Queue[0].DurationMS != 1234 {
		t.Fatalf("unexpected upcoming track: %#v", queue.Queue)
	}
	data, err := json.Marshal(queue.Queue[0])
	if err != nil {
		t.Fatalf("marshal queue item: %v", err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("decode queue item: %v", err)
	}
	if fields["album"] != "Next Album" || fields["duration_ms"] != float64(1234) || fields["explicit"] != false {
		t.Fatalf("missing metadata in JSON: %s", data)
	}
}

package spotify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type staticTokenProvider struct{}

func (staticTokenProvider) Token(ctx context.Context) (Token, error) {
	return Token{AccessToken: "token"}, nil
}

func newTestClient(t *testing.T, handler http.Handler) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client, srv.Close
}

func TestSearchTrack(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/search") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		payload := map[string]any{
			"tracks": map[string]any{
				"items": []map[string]any{{
					"id":            "t1",
					"uri":           "spotify:track:t1",
					"name":          "Song",
					"duration_ms":   123000,
					"explicit":      false,
					"is_playable":   true,
					"album":         map[string]any{"name": "Album"},
					"artists":       []map[string]any{{"name": "Artist"}},
					"external_urls": map[string]string{"spotify": "https://open.spotify.com/track/t1"},
				}},
				"limit":  1,
				"offset": 0,
				"total":  1,
			},
		}
		_ = json.NewEncoder(w).Encode(payload)
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	res, err := client.Search(context.Background(), "track", "song", 1, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].Name != "Song" {
		t.Fatalf("unexpected items: %#v", res.Items)
	}
}

func TestPlaybackNoContent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	status, err := client.Playback(context.Background())
	if err != nil {
		t.Fatalf("playback: %v", err)
	}
	if status.IsPlaying {
		t.Fatalf("expected not playing")
	}
}

func TestPlaybackWithItem(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/me/player":
			payload := playbackResponse{
				IsPlaying:  true,
				ProgressMS: 1000,
				Device:     deviceItem{Name: "Desk"},
				Item:       trackItem{ID: "t1", Name: "Song", Artists: []artistRef{{Name: "Artist"}}},
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	status, err := client.Playback(context.Background())
	if err != nil {
		t.Fatalf("playback: %v", err)
	}
	if status.Item == nil || status.Item.Name != "Song" {
		t.Fatalf("expected item")
	}
}

func TestPlaybackHydratesSparseItem(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/me/player":
			payload := playbackResponse{
				IsPlaying: true,
				Item:      trackItem{ID: "t1", URI: "spotify:track:t1", Name: "Song"},
			}
			_ = json.NewEncoder(w).Encode(payload)
		case "/tracks/t1":
			payload := trackItem{
				ID:      "t1",
				URI:     "spotify:track:t1",
				Name:    "Song",
				Album:   albumRef{Name: "Album"},
				Artists: []artistRef{{Name: "Artist"}},
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	status, err := client.Playback(context.Background())
	if err != nil {
		t.Fatalf("playback: %v", err)
	}
	if status.Item == nil || len(status.Item.Artists) != 1 || status.Item.Artists[0] != "Artist" || status.Item.Album != "Album" {
		t.Fatalf("expected hydrated item: %#v", status.Item)
	}
}

func TestPlaybackKeepsSparseItemWhenHydrationFails(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/player" {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		payload := playbackResponse{
			IsPlaying:  true,
			ProgressMS: 1000,
			Device:     deviceItem{Name: "Desk"},
			Item:       trackItem{ID: "t1", URI: "spotify:track:t1", Name: "Song"},
		}
		_ = json.NewEncoder(w).Encode(payload)
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	status, err := client.Playback(context.Background())
	if err != nil {
		t.Fatalf("playback: %v", err)
	}
	if status.Item == nil || status.Item.Name != "Song" {
		t.Fatalf("expected item")
	}
}

func TestSeekAddsQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("position_ms") != "120000" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	if err := client.Seek(context.Background(), 120000); err != nil {
		t.Fatalf("seek: %v", err)
	}
}

func TestAddTracksPayload(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(data), "spotify:track:t1") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	if err := client.AddTracks(context.Background(), "p1", []string{"spotify:track:t1"}); err != nil {
		t.Fatalf("add tracks: %v", err)
	}
}

func TestPlayContextURI(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "context_uri") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	if err := client.Play(context.Background(), "spotify:album:a1"); err != nil {
		t.Fatalf("play: %v", err)
	}
}

func TestQueueNoContent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	queue, err := client.Queue(context.Background())
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if len(queue.Queue) != 0 {
		t.Fatalf("expected empty queue")
	}
}

func TestQueueWithCurrent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(queueResponse{CurrentlyPlaying: trackItem{ID: "t1", Name: "Song"}})
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	queue, err := client.Queue(context.Background())
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	if queue.CurrentlyPlaying == nil {
		t.Fatalf("expected currently playing")
	}
}

func TestVolumeQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("volume_percent") != "42" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	if err := client.Volume(context.Background(), 42); err != nil {
		t.Fatalf("volume: %v", err)
	}
}

func TestShuffleQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "true" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	if err := client.Shuffle(context.Background(), true); err != nil {
		t.Fatalf("shuffle: %v", err)
	}
}

func TestRepeatQuery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "track" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	client, closeFn := newTestClient(t, handler)
	defer closeFn()
	if err := client.Repeat(context.Background(), "track"); err != nil {
		t.Fatalf("repeat: %v", err)
	}
}

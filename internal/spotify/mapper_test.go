package spotify

import (
	"encoding/json"
	"testing"
)

func TestMapSearchItemTrack(t *testing.T) {
	raw := json.RawMessage(`{"id":"t1","uri":"spotify:track:t1","name":"Song","duration_ms":1000,"explicit":false,"is_playable":true,"album":{"name":"Album"},"artists":[{"name":"Artist"}]}`)
	item, err := mapSearchItem("track", raw)
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	if item.Name != "Song" || item.Type != "track" {
		t.Fatalf("unexpected item: %#v", item)
	}
}

func TestMapArtist(t *testing.T) {
	item := mapArtist(artistItem{ID: "a1", Name: "Artist", Followers: struct {
		Total int `json:"total"`
	}{Total: 10}})
	if item.Followers != 10 {
		t.Fatalf("followers: %d", item.Followers)
	}
}

func TestMapSearchItemUnsupported(t *testing.T) {
	if _, err := mapSearchItem("unknown", json.RawMessage(`{}`)); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMapSearchItemAlbum(t *testing.T) {
	raw := json.RawMessage(`{"id":"a1","uri":"spotify:album:a1","name":"Album","artists":[{"name":"Artist"}],"album_type":"album"}`)
	if _, err := mapSearchItem("album", raw); err != nil {
		t.Fatalf("album: %v", err)
	}
}

func TestMapSearchItemArtist(t *testing.T) {
	raw := json.RawMessage(`{"id":"ar1","uri":"spotify:artist:ar1","name":"Artist","followers":{"total":1}}`)
	if _, err := mapSearchItem("artist", raw); err != nil {
		t.Fatalf("artist: %v", err)
	}
}

func TestMapSearchItemPlaylist(t *testing.T) {
	raw := json.RawMessage(`{"id":"p1","uri":"spotify:playlist:p1","name":"Playlist","tracks":{"total":1},"owner":{"display_name":"Owner"}}`)
	if _, err := mapSearchItem("playlist", raw); err != nil {
		t.Fatalf("playlist: %v", err)
	}
}

func TestMapSearchItemShow(t *testing.T) {
	raw := json.RawMessage(`{"id":"s1","uri":"spotify:show:s1","name":"Show","publisher":"Pub","total_episodes":1}`)
	if _, err := mapSearchItem("show", raw); err != nil {
		t.Fatalf("show: %v", err)
	}
}

func TestMapSearchItemEpisode(t *testing.T) {
	raw := json.RawMessage(`{"id":"e1","uri":"spotify:episode:e1","name":"Episode","duration_ms":1000}`)
	if _, err := mapSearchItem("episode", raw); err != nil {
		t.Fatalf("episode: %v", err)
	}
}

func TestMapSearchItemBadJSON(t *testing.T) {
	if _, err := mapSearchItem("track", json.RawMessage(`{`)); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMapSearchItemBadJSONOtherKinds(t *testing.T) {
	kinds := []string{"album", "artist", "playlist", "show", "episode"}
	for _, kind := range kinds {
		if _, err := mapSearchItem(kind, json.RawMessage(`{`)); err == nil {
			t.Fatalf("expected error for %s", kind)
		}
	}
}

func TestMapPlaylistTotalTracks(t *testing.T) {
	var owned playlistItem
	if err := json.Unmarshal([]byte(`{"id":"p1","items":{"total":31}}`), &owned); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := mapPlaylist(owned).TotalTracks; got != 31 {
		t.Fatalf("TotalTracks = %d, want 31 from items.total", got)
	}

	// Spotify omits the count for playlists the user neither owns nor
	// collaborates on.
	var foreign playlistItem
	if err := json.Unmarshal([]byte(`{"id":"p2"}`), &foreign); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := mapPlaylist(foreign).TotalTracks; got != 0 {
		t.Fatalf("TotalTracks = %d, want 0 when the count is absent", got)
	}

	// Extended Quota Mode apps still receive the pre-2026 container name.
	var legacy playlistItem
	if err := json.Unmarshal([]byte(`{"id":"p3","tracks":{"total":7}}`), &legacy); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := mapPlaylist(legacy).TotalTracks; got != 7 {
		t.Fatalf("TotalTracks = %d, want 7 from tracks.total", got)
	}

	var both playlistItem
	if err := json.Unmarshal([]byte(`{"id":"p4","items":{"total":31},"tracks":{"total":7}}`), &both); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := mapPlaylist(both).TotalTracks; got != 31 {
		t.Fatalf("TotalTracks = %d, want items.total to win", got)
	}
	if err := json.Unmarshal([]byte(`{"items":{"total":0},"tracks":{"total":7}}`), &both); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := mapPlaylist(both).TotalTracks; got != 0 {
		t.Fatalf("TotalTracks = %d, want an explicit empty items container to win", got)
	}
}

func TestExternalURL(t *testing.T) {
	if externalURL(nil) != "" {
		t.Fatalf("expected empty")
	}
	url := externalURL(map[string]string{"foo": "bar"})
	if url != "bar" {
		t.Fatalf("expected fallback url")
	}
}

package spotify

import "testing"

func TestPlaylistContentPreservesRepeatedTracks(t *testing.T) {
	entries := make([]any, 0, 3)
	for _, id := range []string{"a", "b", "a"} {
		entries = append(entries, map[string]any{"itemV2": map[string]any{"data": map[string]any{"uri": "spotify:track:" + id, "name": id}}})
	}
	payload := map[string]any{"data": map[string]any{"playlistV2": map[string]any{"content": map[string]any{"items": entries, "totalCount": float64(3)}}}}
	items, total := extractPlaylistContentItems(payload, "track")
	if len(items) != 3 || total != 3 {
		t.Fatalf("items=%+v total=%d; repeated positions must be retained", items, total)
	}
	for i, id := range []string{"a", "b", "a"} {
		if items[i].ID != id {
			t.Fatalf("position %d = %q, want %q", i, items[i].ID, id)
		}
	}
}

func TestLibraryContentStillDeduplicatesEntities(t *testing.T) {
	entries := make([]any, 0, 3)
	for _, id := range []string{"a", "b", "a"} {
		entries = append(entries, map[string]any{"item": map[string]any{"data": map[string]any{"uri": "spotify:playlist:" + id, "name": id}}})
	}
	for _, tc := range []struct {
		name    string
		present bool
		total   int
		want    int
	}{
		{"missing", false, 0, 2},
		{"zero", true, 0, 2},
		{"advertised", true, 3, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			library := map[string]any{"items": entries}
			if tc.present {
				library["totalCount"] = float64(tc.total)
			}
			payload := map[string]any{"data": map[string]any{"me": map[string]any{"libraryV3": library}}}
			items, total := extractLibraryV3Items(payload, "playlist")
			if len(items) != 2 || total != tc.want || items[0].ID != "a" || items[1].ID != "b" {
				t.Fatalf("library set semantics changed: %+v total=%d, want %d", items, total, tc.want)
			}
		})
	}
}

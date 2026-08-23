package spotify

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestExtractItem(t *testing.T) {
	raw := map[string]any{
		"uri":  "spotify:track:abc",
		"name": "Song",
		"artists": []any{
			map[string]any{"name": "Artist"},
		},
		"album": map[string]any{"name": "Album"},
	}
	item, ok := extractItem(raw, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	if item.ID != "abc" || item.Name != "Song" || item.Album != "Album" {
		t.Fatalf("unexpected item: %#v", item)
	}
	if len(item.Artists) != 1 || item.Artists[0] != "Artist" {
		t.Fatalf("unexpected artists: %#v", item.Artists)
	}
}

func TestExtractFetchLibraryTracks(t *testing.T) {
	payload := map[string]any{
		"data": map[string]any{"me": map[string]any{"library": map[string]any{"tracks": map[string]any{
			"totalCount": 2,
			"items": []any{
				map[string]any{"track": map[string]any{
					"_uri": "spotify:track:t1",
					"data": map[string]any{"name": "Song One"},
				}},
				map[string]any{"track": map[string]any{
					"_uri": "spotify:track:t2",
					"data": map[string]any{"name": "Song Two"},
				}},
			},
		}}}},
	}
	items, total, err := extractFetchLibraryTracks(payload)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("expected 2 items, got %d (total %d)", len(items), total)
	}
	if items[0].ID != "t1" || items[0].Name != "Song One" {
		t.Fatalf("unexpected first item: %#v", items[0])
	}
	if items[1].ID != "t2" || items[1].Name != "Song Two" {
		t.Fatalf("unexpected second item: %#v", items[1])
	}
}

func TestExtractFetchLibraryTracksDedupes(t *testing.T) {
	payload := map[string]any{
		"data": map[string]any{"me": map[string]any{"library": map[string]any{"tracks": map[string]any{
			"totalCount": 1,
			"items": []any{
				map[string]any{"track": map[string]any{
					"_uri": "spotify:track:t1",
					"data": map[string]any{"name": "Song"},
				}},
				map[string]any{"track": map[string]any{
					"_uri": "spotify:track:t1",
					"data": map[string]any{"name": "Song"},
				}},
			},
		}}}},
	}
	items, _, err := extractFetchLibraryTracks(payload)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 deduped item, got %d", len(items))
	}
}

func TestExtractFetchLibraryTracksMissingPath(t *testing.T) {
	items, total, err := extractFetchLibraryTracks(map[string]any{})
	if err == nil {
		t.Fatalf("expected error, got %d items (total %d)", len(items), total)
	}
}

func TestExtractFetchLibraryTracksSkipsMalformed(t *testing.T) {
	payload := map[string]any{
		"data": map[string]any{"me": map[string]any{"library": map[string]any{"tracks": map[string]any{
			"totalCount": 0,
			"items": []any{
				"not a map",
				map[string]any{"track": "not a map"},
				map[string]any{"track": map[string]any{"_uri": "spotify:track:t1"}},
				map[string]any{"track": map[string]any{
					"_uri": "spotify:track:t2",
					"data": map[string]any{"name": "Valid"},
				}},
			},
		}}}},
	}
	items, _, err := extractFetchLibraryTracks(payload)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(items) != 1 || items[0].ID != "t2" {
		t.Fatalf("expected 1 valid item, got %#v", items)
	}
}

func TestSearchPaths(t *testing.T) {
	paths := searchPaths("track")
	if len(paths) == 0 {
		t.Fatalf("expected paths")
	}
}

func TestExtractItemFallbacks(t *testing.T) {
	raw := map[string]any{
		"id":    "t1",
		"title": "Song",
		"artists": []any{
			map[string]any{"name": "Artist"},
			map[string]any{"name": "Artist"},
		},
		"albumOfTrack": map[string]any{"name": "Album"},
		"owner":        map[string]any{"name": "Owner"},
		"durationMs":   1200,
		"explicit":     true,
		"total":        5,
		"releaseDate":  "2020-01-01",
		"description":  "desc",
		"isPlayable":   true,
	}
	item, ok := extractItem(raw, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	if item.URI != "spotify:track:t1" || item.Name != "Song" {
		t.Fatalf("unexpected item: %#v", item)
	}
	if len(item.Artists) != 1 || item.Artists[0] != "Artist" {
		t.Fatalf("expected deduped artists: %#v", item.Artists)
	}
	if item.Album != "Album" || item.Owner != "Owner" || item.DurationMS != 1200 {
		t.Fatalf("unexpected fields: %#v", item)
	}
}

func TestExtractItemPathfinderTrackMetadata(t *testing.T) {
	raw := map[string]any{
		"id":   "t1",
		"name": "Song",
		"contentRating": map[string]any{
			"label": "EXPLICIT",
		},
		"duration": map[string]any{
			"totalMilliseconds": 321613,
		},
		"playability": map[string]any{
			"playable": true,
		},
	}
	item, ok := extractItem(raw, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	if !item.Explicit || !item.ExplicitKnown {
		t.Fatalf("expected explicit metadata: %#v", item)
	}
	if item.DurationMS != 321613 {
		t.Fatalf("unexpected duration: %#v", item)
	}
	if !item.IsPlayable {
		t.Fatalf("expected playable metadata: %#v", item)
	}
}

func TestExtractItemPathfinderTrackDurationFallback(t *testing.T) {
	raw := map[string]any{
		"id": "t1",
		"trackDuration": map[string]any{
			"totalMilliseconds": 1200,
		},
	}
	item, ok := extractItem(raw, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	if item.DurationMS != 1200 {
		t.Fatalf("unexpected duration: %#v", item)
	}
}

func TestItemJSONIncludesKnownExplicitFalse(t *testing.T) {
	raw := map[string]any{
		"id": "t1",
		"contentRating": map[string]any{
			"label": "NONE",
		},
	}
	item, ok := extractItem(raw, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	explicit, ok := fields["explicit"]
	if !ok {
		t.Fatalf("expected explicit false in JSON: %s", encoded)
	}
	var value bool
	if err := json.Unmarshal(explicit, &value); err != nil {
		t.Fatalf("unmarshal explicit: %v", err)
	}
	if value {
		t.Fatalf("expected explicit false in JSON: %s", encoded)
	}
}

func TestItemJSONOmitsUnknownExplicitFalse(t *testing.T) {
	encoded, err := json.Marshal(Item{ID: "p1", URI: "spotify:playlist:p1", Type: "playlist"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := fields["explicit"]; ok {
		t.Fatalf("unexpected explicit false in JSON: %s", encoded)
	}
}

func TestExtractItemArtistsContainers(t *testing.T) {
	raw := map[string]any{
		"uri":  "spotify:track:abc",
		"name": "Song",
		"artists": map[string]any{
			"items": []any{
				map[string]any{"name": "Artist One"},
				map[string]any{"name": "Artist Two"},
			},
		},
	}
	item, ok := extractItem(raw, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	if len(item.Artists) != 2 || item.Artists[0] != "Artist One" || item.Artists[1] != "Artist Two" {
		t.Fatalf("unexpected artists: %#v", item.Artists)
	}
}

func TestExtractItemArtistsEdges(t *testing.T) {
	raw := map[string]any{
		"uri":  "spotify:track:abc",
		"name": "Song",
		"artists": map[string]any{
			"edges": []any{
				map[string]any{"node": map[string]any{"name": "Artist One"}},
				map[string]any{"node": map[string]any{"name": "Artist Two"}},
			},
		},
	}
	item, ok := extractItem(raw, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	if len(item.Artists) != 2 || item.Artists[0] != "Artist One" || item.Artists[1] != "Artist Two" {
		t.Fatalf("unexpected artists: %#v", item.Artists)
	}
}

func TestExtractItemFirstArtistItems(t *testing.T) {
	raw := map[string]any{
		"uri":  "spotify:track:abc",
		"name": "Song",
		"firstArtist": map[string]any{
			"items": []any{
				map[string]any{"profile": map[string]any{"name": "Artist One"}},
			},
		},
	}
	item, ok := extractItem(raw, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	if len(item.Artists) != 1 || item.Artists[0] != "Artist One" {
		t.Fatalf("unexpected artists: %#v", item.Artists)
	}
}

func TestExtractItemOtherArtistsItems(t *testing.T) {
	raw := map[string]any{
		"uri":  "spotify:track:abc",
		"name": "Song",
		"firstArtist": map[string]any{
			"items": []any{
				map[string]any{"profile": map[string]any{"name": "Artist One"}},
			},
		},
		"otherArtists": map[string]any{
			"items": []any{
				map[string]any{"profile": map[string]any{"name": "Artist Two"}},
			},
		},
	}
	item, ok := extractItem(raw, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	if len(item.Artists) != 2 || item.Artists[0] != "Artist One" || item.Artists[1] != "Artist Two" {
		t.Fatalf("unexpected artists: %#v", item.Artists)
	}
}

func TestExtractSearchTrackItemNestedArtists(t *testing.T) {
	payload := map[string]any{
		"data": map[string]any{
			"searchV2": map[string]any{
				"tracksV2": map[string]any{
					"totalCount": 1,
					"items": []any{
						map[string]any{
							"item": map[string]any{
								"data": map[string]any{
									"uri":  "spotify:track:abc",
									"name": "Song",
									"artists": map[string]any{
										"items": []any{
											map[string]any{"profile": map[string]any{"name": "Artist"}},
										},
									},
									"albumOfTrack": map[string]any{"name": "Album"},
								},
							},
						},
					},
				},
			},
		},
	}
	items, total := extractSearchItems(payload, "track")
	if total != 1 || len(items) != 1 {
		t.Fatalf("unexpected items: %#v total=%d", items, total)
	}
	if len(items[0].Artists) != 1 || items[0].Artists[0] != "Artist" || items[0].Album != "Album" {
		t.Fatalf("unexpected metadata: %#v", items[0])
	}
}

func TestExtractPlaylistDoesNotLeakNestedTrackAlbum(t *testing.T) {
	payload := map[string]any{
		"data": map[string]any{
			"playlistV2": map[string]any{
				"uri":  "spotify:playlist:p1",
				"name": "Playlist",
				"content": map[string]any{
					"items": []any{
						map[string]any{"itemV2": map[string]any{"data": map[string]any{
							"track": map[string]any{
								"uri":   "spotify:track:t1",
								"name":  "Song",
								"album": map[string]any{"name": "Album"},
							},
						}}},
					},
				},
			},
		},
	}
	item, ok := extractItemFromPayload(payload, "playlist")
	if !ok {
		t.Fatalf("expected playlist")
	}
	if item.Album != "" {
		t.Fatalf("playlist leaked album metadata: %#v", item)
	}
}

func TestExtractItemFromPayloadPrefersTrackUnion(t *testing.T) {
	payload := map[string]any{
		"data": map[string]any{
			"trackUnion": map[string]any{
				"uri":  "spotify:track:primary",
				"name": "Primary",
				"artists": []any{
					map[string]any{"name": "Main Artist"},
				},
			},
			"track": map[string]any{
				"uri":  "spotify:track:secondary",
				"name": "Secondary",
				"artists": []any{
					map[string]any{"name": "Wrong Artist"},
				},
			},
			"other": map[string]any{
				"items": []any{
					map[string]any{
						"uri":  "spotify:track:secondary",
						"name": "Secondary",
						"artists": []any{
							map[string]any{"name": "Wrong Artist"},
						},
					},
				},
			},
		},
	}
	item, ok := extractItemFromPayload(payload, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	if item.ID != "primary" || len(item.Artists) != 1 || item.Artists[0] != "Main Artist" {
		t.Fatalf("unexpected item: %#v", item)
	}
}

func TestExtractItemArtistsIDName(t *testing.T) {
	raw := map[string]any{
		"uri":  "spotify:track:abc",
		"name": "Song",
		"artists": []any{
			map[string]any{"id": "ar1", "name": "Artist One"},
		},
	}
	item, ok := extractItem(raw, "track")
	if !ok {
		t.Fatalf("expected item")
	}
	if len(item.Artists) != 1 || item.Artists[0] != "Artist One" {
		t.Fatalf("unexpected artists: %#v", item.Artists)
	}
}

func TestExtractSearchItemsFallback(t *testing.T) {
	payload := map[string]any{
		"data": map[string]any{
			"nested": map[string]any{
				"item": map[string]any{"uri": "spotify:track:abc", "name": "Song"},
			},
		},
	}
	items, total := extractSearchItems(payload, "track")
	if len(items) == 0 || items[0].URI != "spotify:track:abc" {
		t.Fatalf("unexpected items: %#v", items)
	}
	if total != len(items) {
		t.Fatalf("expected total to match items")
	}
}

func TestSearchPathsVariants(t *testing.T) {
	if len(searchPaths("album")) == 0 {
		t.Fatalf("expected album paths")
	}
	if len(searchPaths("artist")) == 0 {
		t.Fatalf("expected artist paths")
	}
	if len(searchPaths("playlist")) == 0 {
		t.Fatalf("expected playlist paths")
	}
	if len(searchPaths("show")) == 0 {
		t.Fatalf("expected show paths")
	}
	if len(searchPaths("episode")) == 0 {
		t.Fatalf("expected episode paths")
	}
	if searchPaths("unknown") != nil {
		t.Fatalf("expected nil paths")
	}
}

func TestHelperFinds(t *testing.T) {
	value := map[string]any{
		"nested": []any{
			map[string]any{"title": "Hello"},
			map[string]any{"uri": "spotify:track:abc"},
		},
	}
	if name := findFirstName(value); name != "Hello" {
		t.Fatalf("unexpected name: %s", name)
	}
	if uri := findFirstURI(value, "track"); uri != "spotify:track:abc" {
		t.Fatalf("unexpected uri: %s", uri)
	}
	payload := map[string]any{"a": map[string]any{"b": map[string]any{"value": "ok", "num": 2.0, "flag": true}}}
	if m, ok := getMap(payload, "a", "b"); !ok || getString(m, "value") != "ok" {
		t.Fatalf("unexpected map")
	}
	m, _ := getMap(payload, "a", "b")
	if getInt(m, "num") != 2 || !getBool(m, "flag") {
		t.Fatalf("unexpected helpers")
	}
	if idFromURI("abc") != "abc" {
		t.Fatalf("unexpected id")
	}
	if typeFromURI("abc") != "" {
		t.Fatalf("unexpected type")
	}
}

func TestVisitItemsSlice(t *testing.T) {
	values := []any{
		map[string]any{"uri": "spotify:track:abc", "name": "Song"},
	}
	items := []Item{}
	visitItems(values, "track", &items)
	if len(items) != 1 || items[0].ID != "abc" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestHelperNilAndOwnerFallback(t *testing.T) {
	if getString(nil, "key") != "" {
		t.Fatalf("expected empty string")
	}
	if getInt(nil, "key") != 0 {
		t.Fatalf("expected zero int")
	}
	if getBool(nil, "key") {
		t.Fatalf("expected false bool")
	}
	value := map[string]any{
		"user": map[string]any{"name": "Owner"},
	}
	if name := extractOwnerName(value); name != "Owner" {
		t.Fatalf("unexpected owner: %s", name)
	}
	container := map[string]any{
		"item": map[string]any{"uri": "spotify:track:q1", "name": "Song"},
	}
	items := extractItemsFromContainer(container, "track")
	if len(items) == 0 || items[0].ID != "q1" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestExtractRequestedItemFromPayloadMatchesRequestedEntity(t *testing.T) {
	tests := []struct {
		name      string
		kind      string
		key       string
		entity    map[string]any
		wantName  string
		followers int
		genres    []string
	}{
		{
			name: "artist profile wins over related playlist",
			kind: "artist",
			key:  "artistUnion",
			entity: map[string]any{
				"uri":     "spotify:artist:target",
				"profile": map[string]any{"name": "Weezer"},
				"stats":   map[string]any{"followers": float64(123456)},
				"genres":  map[string]any{"items": []any{map[string]any{"name": "alternative rock"}, map[string]any{"name": "power pop"}}},
				"relatedContent": map[string]any{
					"playlist": map[string]any{"name": "Weezer, The Shins, Silversun Pickups"},
				},
			},
			wantName:  "Weezer",
			followers: 123456,
			genres:    []string{"alternative rock", "power pop"},
		},
		{
			name:     "album union ignores nested artist album",
			kind:     "album",
			key:      "albumUnion",
			entity:   map[string]any{"uri": "spotify:album:target", "name": "Target Album"},
			wantName: "Target Album",
		},
		{
			name:     "podcast union ignores related show",
			kind:     "show",
			key:      "podcastUnionV2",
			entity:   map[string]any{"uri": "spotify:show:target", "name": "Target Show"},
			wantName: "Target Show",
		},
		{
			name:     "episode union ignores related episode",
			kind:     "episode",
			key:      "episodeUnionV2",
			entity:   map[string]any{"uri": "spotify:episode:target", "name": "Target Episode"},
			wantName: "Target Episode",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := map[string]any{"data": map[string]any{
				"related": map[string]any{
					"uri":  "spotify:" + test.kind + ":other",
					"name": "Unrelated " + test.kind,
				},
				test.key: test.entity,
			}}
			item, ok := extractRequestedItemFromPayload(payload, test.kind, "spotify:"+test.kind+":target")
			if !ok {
				t.Fatal("expected requested entity")
			}
			if item.ID != "target" || item.Name != test.wantName || item.Followers != test.followers || !reflect.DeepEqual(item.Genres, test.genres) {
				t.Fatalf("unexpected requested entity: %#v", item)
			}
		})
	}
}

func TestExtractRequestedItemFromPayloadRejectsDifferentEntity(t *testing.T) {
	payload := map[string]any{"data": map[string]any{"artistUnion": map[string]any{
		"uri":     "spotify:artist:other",
		"profile": map[string]any{"name": "Other Artist"},
	}}}
	if item, ok := extractRequestedItemFromPayload(payload, "artist", "spotify:artist:target"); ok {
		t.Fatalf("returned unrelated artist: %#v", item)
	}
}

func TestExtractRequestedItemFromPayloadFindsNestedMatchingEntity(t *testing.T) {
	payload := map[string]any{"data": map[string]any{"unexpectedContainer": map[string]any{
		"entity": map[string]any{"uri": "spotify:show:target", "name": "Nested Show"},
	}}}
	item, ok := extractRequestedItemFromPayload(payload, "show", "spotify:show:target")
	if !ok || item.Name != "Nested Show" {
		t.Fatalf("unexpected fallback entity: %#v, found=%t", item, ok)
	}
}

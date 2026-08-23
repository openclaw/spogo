package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestExtractSearchItemsRealPathfinderShapes(t *testing.T) {
	const fixture = `{
		"data":{"searchV2":{
			"albumsV2":{"totalCount":899,"items":[{"__typename":"AlbumResponseWrapper","data":{
				"uri":"spotify:album:65sHj9PvsbyD0uugGHjueN","name":"Weezer (Teal Album)",
				"artists":{"items":[{"uri":"spotify:artist:3jOstUTkEu2JkjvRdBA5Gu","profile":{"name":"Weezer"}}]},
				"date":{"year":2019}
			}}]},
			"artists":{"totalCount":829,"items":[{"__typename":"ArtistResponseWrapper","data":{
				"uri":"spotify:artist:3jOstUTkEu2JkjvRdBA5Gu","profile":{"name":"Weezer"}
			}}]},
			"playlists":{"totalCount":911,"items":[{"__typename":"PlaylistResponseWrapper","data":{
				"uri":"spotify:playlist:0Q3ugz23LAXFg2PvXJ8hMx","name":"TRANCE 2026",
				"ownerV2":{"data":{"name":"YOU LOVE DANCE","uri":"spotify:user:1190272463"}}
			}}]},
			"podcasts":{"totalCount":857,"items":[{"__typename":"PodcastResponseWrapper","data":{
				"uri":"spotify:show:4XPl3uEEL9hvqMkoZrzbx5","name":"Darknet Diaries",
				"publisher":{"name":"Jack Rhysider"}
			}}]},
			"episodes":{"totalCount":1000,"items":[{"__typename":"EpisodeResponseWrapper","data":{
				"uri":"spotify:episode:6CQAC1k7sUVk8FQsXABlRU","name":"178: Ubiquiti",
				"duration":{"totalMilliseconds":2429088},"releaseDate":{"isoString":"2026-08-04T07:00:00Z"},
				"podcastV2":{"data":{"name":"Darknet Diaries","uri":"spotify:show:4XPl3uEEL9hvqMkoZrzbx5"}}
			}}]}
		}}
	}`
	var payload map[string]any
	if err := json.Unmarshal([]byte(fixture), &payload); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		kind  string
		total int
		check func(*testing.T, Item)
	}{
		{"album", 899, func(t *testing.T, item Item) {
			if item.Name != "Weezer (Teal Album)" || !reflect.DeepEqual(item.Artists, []string{"Weezer"}) || item.ReleaseDate != "2019" {
				t.Fatalf("album metadata: %#v", item)
			}
		}},
		{"artist", 829, func(t *testing.T, item Item) {
			if item.Name != "Weezer" || item.ID != "3jOstUTkEu2JkjvRdBA5Gu" {
				t.Fatalf("artist metadata: %#v", item)
			}
		}},
		{"playlist", 911, func(t *testing.T, item Item) {
			if item.Name != "TRANCE 2026" || item.Owner != "YOU LOVE DANCE" {
				t.Fatalf("playlist metadata: %#v", item)
			}
		}},
		{"show", 857, func(t *testing.T, item Item) {
			if item.Name != "Darknet Diaries" || item.Publisher != "Jack Rhysider" {
				t.Fatalf("show metadata: %#v", item)
			}
		}},
		{"episode", 1000, func(t *testing.T, item Item) {
			if item.Name != "178: Ubiquiti" || item.Show != "Darknet Diaries" || item.DurationMS != 2429088 || item.ReleaseDate != "2026-08-04" {
				t.Fatalf("episode metadata: %#v", item)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.kind, func(t *testing.T) {
			items, total := extractSearchItems(payload, test.kind)
			if total != test.total || len(items) != 1 {
				t.Fatalf("items=%#v total=%d", items, total)
			}
			test.check(t, items[0])
		})
	}
}

func TestExtractItemRealInfoMetadata(t *testing.T) {
	tests := []struct {
		name  string
		kind  string
		raw   string
		check func(*testing.T, Item)
	}{
		{"album", "album", `{"uri":"spotify:album:a1","name":"Parachutes","date":{"isoString":"2000-07-10T00:00:00Z"},"tracksV2":{"totalCount":10}}`, func(t *testing.T, item Item) {
			if item.ReleaseDate != "2000-07-10" || item.TotalTracks != 10 {
				t.Fatalf("album: %#v", item)
			}
		}},
		{"artist", "artist", `{"uri":"spotify:artist:a1","profile":{"name":"Weezer"},"stats":{"followers":5567807}}`, func(t *testing.T, item Item) {
			if item.Name != "Weezer" || item.Followers != 5567807 {
				t.Fatalf("artist: %#v", item)
			}
		}},
		{"playlist", "playlist", `{"uri":"spotify:playlist:p1","name":"TRANCE 2026","ownerV2":{"data":{"name":"YOU LOVE DANCE"}},"content":{"totalCount":197},"followers":352874}`, func(t *testing.T, item Item) {
			if item.Owner != "YOU LOVE DANCE" || item.TotalTracks != 197 || item.Followers != 352874 {
				t.Fatalf("playlist: %#v", item)
			}
		}},
		{"show", "show", `{"uri":"spotify:show:s1","name":"Darknet Diaries","episodesV2":{"totalCount":227}}`, func(t *testing.T, item Item) {
			if item.TotalEpisodes != 227 {
				t.Fatalf("show: %#v", item)
			}
		}},
		{"wrapped URI", "artist", `{"_uri":"spotify:artist:a1","data":{"profile":{"name":"Artist"}}}`, func(t *testing.T, item Item) {
			if item.URI != "spotify:artist:a1" || item.Name != "Artist" {
				t.Fatalf("wrapped artist: %#v", item)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var raw map[string]any
			if err := json.Unmarshal([]byte(test.raw), &raw); err != nil {
				t.Fatal(err)
			}
			item, ok := extractItem(raw, test.kind)
			if !ok {
				t.Fatalf("item not extracted: %#v", raw)
			}
			test.check(t, item)
		})
	}
}

func TestConnectSearchHydratesArtistFollowersWithoutWebAPI(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Query().Get("operationName") {
		case "searchDesktop":
			return jsonResponse(http.StatusOK, map[string]any{"data": map[string]any{"searchV2": map[string]any{"artists": map[string]any{
				"totalCount": 1, "items": []any{map[string]any{"data": map[string]any{"uri": "spotify:artist:a1", "profile": map[string]any{"name": "Weezer"}}}},
			}}}}), nil
		case "queryArtistOverview":
			return jsonResponse(http.StatusOK, map[string]any{"data": map[string]any{"artistUnion": map[string]any{
				"uri": "spotify:artist:a1", "profile": map[string]any{"name": "Weezer"}, "stats": map[string]any{"followers": 5567807},
			}}}), nil
		default:
			return nil, fmt.Errorf("unexpected operation or public API request: %s", req.URL)
		}
	})
	client := newConnectClientForTests(transport)
	client.hashes.hashes["searchDesktop"] = "hash"
	client.hashes.hashes["queryArtistOverview"] = "hash"
	result, err := client.Search(context.Background(), "artist", "weezer", 1, 0)
	if err != nil || len(result.Items) != 1 || result.Items[0].Followers != 5567807 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestConnectSearchHydratesPlaylistAndShowCounts(t *testing.T) {
	tests := []struct {
		kind      string
		container string
		operation string
		key       string
		entity    map[string]any
		check     func(Item) bool
	}{
		{"playlist", "playlists", "fetchPlaylist", "playlistV2", map[string]any{"uri": "spotify:playlist:p1", "name": "List", "ownerV2": map[string]any{"data": map[string]any{"name": "Owner"}}, "content": map[string]any{"totalCount": 197}}, func(item Item) bool { return item.TotalTracks == 197 && item.Owner == "Owner" }},
		{"show", "podcasts", "queryPodcastEpisodes", "podcastUnionV2", map[string]any{"uri": "spotify:show:s1", "name": "Show", "episodesV2": map[string]any{"totalCount": 227}}, func(item Item) bool { return item.TotalEpisodes == 227 && item.Publisher == "Publisher" }},
		{"album", "albumsV2", "getAlbum", "albumUnion", map[string]any{"uri": "spotify:album:a1", "name": "Album", "tracksV2": map[string]any{"totalCount": 10}, "date": map[string]any{"isoString": "2000-07-10T00:00:00Z"}}, func(item Item) bool { return item.TotalTracks == 10 && item.ReleaseDate == "2000-07-10" }},
	}
	for _, test := range tests {
		t.Run(test.kind, func(t *testing.T) {
			transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				switch req.URL.Query().Get("operationName") {
				case "searchDesktop":
					entity := map[string]any{"uri": test.entity["uri"], "name": test.entity["name"]}
					if test.kind == "show" {
						entity["publisher"] = map[string]any{"name": "Publisher"}
					}
					return jsonResponse(http.StatusOK, map[string]any{"data": map[string]any{"searchV2": map[string]any{test.container: map[string]any{
						"totalCount": 1, "items": []any{map[string]any{"data": entity}},
					}}}}), nil
				case test.operation:
					return jsonResponse(http.StatusOK, map[string]any{"data": map[string]any{test.key: test.entity}}), nil
				default:
					return nil, fmt.Errorf("unexpected operation %q", req.URL.Query().Get("operationName"))
				}
			})
			client := newConnectClientForTests(transport)
			client.hashes.hashes["searchDesktop"] = "hash"
			client.hashes.hashes[test.operation] = "hash"
			result, err := client.Search(context.Background(), test.kind, "query", 1, 0)
			if err != nil || len(result.Items) != 1 || !test.check(result.Items[0]) {
				t.Fatalf("result=%#v err=%v", result, err)
			}
		})
	}
}

func TestConnectClientUsesSharedDefaultTimeout(t *testing.T) {
	client, err := NewConnectClient(ConnectOptions{Source: cookieSourceStub{}})
	if err != nil {
		t.Fatal(err)
	}
	if client.client.Timeout != defaultHTTPClientTimeout {
		t.Fatalf("timeout=%s want=%s", client.client.Timeout, defaultHTTPClientTimeout)
	}
}

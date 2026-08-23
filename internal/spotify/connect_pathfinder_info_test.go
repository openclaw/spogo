package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConnectLibraryV3Helpers(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Query().Get("operationName") {
		case "libraryV3":
			query := req.URL.Query()
			variables := query.Get("variables")
			var decoded map[string]any
			if err := json.Unmarshal([]byte(variables), &decoded); err != nil {
				t.Fatalf("unmarshal libraryV3 variables: %v", err)
			}
			if features, ok := decoded["features"].([]any); !ok || len(features) != 0 {
				t.Fatalf("libraryV3 must not request pseudo-playlists: %#v", decoded["features"])
			}
			switch {
			case strings.Contains(variables, `"Playlists"`):
				return jsonResponse(http.StatusOK, map[string]any{
					"data": map[string]any{"me": map[string]any{"libraryV3": map[string]any{
						"totalCount": 1,
						"items": []any{map[string]any{"item": map[string]any{"data": map[string]any{
							"uri":         "spotify:playlist:p1",
							"name":        "Playlist",
							"ownerV2":     map[string]any{"data": map[string]any{"name": "Owner"}},
							"totalTracks": 12,
						}}}},
					}}},
				}), nil
			case strings.Contains(variables, `"Albums"`):
				return jsonResponse(http.StatusOK, map[string]any{
					"data": map[string]any{"me": map[string]any{"libraryV3": map[string]any{
						"totalCount": 1,
						"items": []any{map[string]any{"item": map[string]any{"data": map[string]any{
							"uri":  "spotify:album:a1",
							"name": "Album",
						}}}},
					}}},
				}), nil
			case strings.Contains(variables, `"Artists"`):
				return jsonResponse(http.StatusOK, map[string]any{
					"data": map[string]any{"me": map[string]any{"libraryV3": map[string]any{
						"totalCount": 2,
						"items": []any{map[string]any{"item": map[string]any{"data": map[string]any{
							"uri": "spotify:artist:ar1", "profile": map[string]any{"name": "Artist"},
						}}}},
					}}},
				}), nil
			}
		case "fetchLibraryTracks":
			var vars map[string]any
			if err := json.Unmarshal([]byte(req.URL.Query().Get("variables")), &vars); err != nil {
				t.Fatalf("unmarshal fetchLibraryTracks variables: %v", err)
			}
			if got := getString(vars, "uri"); got != "spotify:collection:tracks" {
				t.Fatalf("fetchLibraryTracks uri = %q", got)
			}
			if got := getInt(vars, "limit"); got != 10 {
				t.Fatalf("fetchLibraryTracks limit = %d", got)
			}
			if got := getInt(vars, "offset"); got != 0 {
				t.Fatalf("fetchLibraryTracks offset = %d", got)
			}
			return jsonResponse(http.StatusOK, map[string]any{
				"data": map[string]any{"me": map[string]any{"library": map[string]any{"tracks": map[string]any{
					"totalCount": 1,
					"items": []any{map[string]any{"track": map[string]any{
						"_uri": "spotify:track:t1",
						"data": map[string]any{
							"name": "Song",
						},
					}}},
				}}}},
			}), nil
		case "fetchPlaylist":
			return jsonResponse(http.StatusOK, map[string]any{
				"data": map[string]any{"playlistV2": map[string]any{"content": map[string]any{
					"totalCount": 1,
					"items": []any{map[string]any{"itemV2": map[string]any{"data": map[string]any{
						"track": map[string]any{"uri": "spotify:track:t1", "name": "Song"},
					}}}},
				}}},
			}), nil
		}
		return textResponse(http.StatusNotFound, "missing"), nil
	})
	client := newConnectClientForTests(transport)
	for _, op := range []string{"libraryV3", "fetchPlaylist", "fetchLibraryTracks"} {
		client.hashes.hashes[op] = "hash"
	}

	playlists, total, err := client.playlists(context.Background(), 10, 0)
	if err != nil || total != 1 || len(playlists) != 1 || playlists[0].ID != "p1" {
		t.Fatalf("playlists: items=%#v total=%d err=%v", playlists, total, err)
	}
	tracks, total, err := client.playlistTracks(context.Background(), "p1", 10, 0)
	if err != nil || total != 1 || len(tracks) != 1 || tracks[0].ID != "t1" {
		t.Fatalf("playlist tracks: items=%#v total=%d err=%v", tracks, total, err)
	}
	libraryTracks, total, err := client.libraryTracks(context.Background(), 10, 0)
	if err != nil || total != 1 || len(libraryTracks) != 1 || libraryTracks[0].ID != "t1" {
		t.Fatalf("library tracks: items=%#v total=%d err=%v", libraryTracks, total, err)
	}
	albums, total, err := client.libraryAlbums(context.Background(), 10, 0)
	if err != nil || total != 1 || len(albums) != 1 || albums[0].ID != "a1" {
		t.Fatalf("library albums: items=%#v total=%d err=%v", albums, total, err)
	}
	artists, total, next, err := client.FollowedArtists(context.Background(), 1, "")
	if err != nil || total != 2 || len(artists) != 1 || artists[0].Name != "Artist" || next != "ar1" {
		t.Fatalf("followed artists: items=%#v total=%d next=%q err=%v", artists, total, next, err)
	}
}

func TestConnectFollowedArtistsCursorUsesLibraryV3(t *testing.T) {
	requests := 0
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Query().Get("operationName") != "libraryV3" {
			t.Fatalf("unexpected request %s", req.URL)
		}
		var vars map[string]any
		if err := json.Unmarshal([]byte(req.URL.Query().Get("variables")), &vars); err != nil {
			t.Fatal(err)
		}
		artists := []any{
			map[string]any{"item": map[string]any{"data": map[string]any{"uri": "spotify:artist:a1", "profile": map[string]any{"name": "First"}}}},
			map[string]any{"item": map[string]any{"data": map[string]any{"uri": "spotify:artist:a2", "profile": map[string]any{"name": "Second"}}}},
			map[string]any{"item": map[string]any{"data": map[string]any{"uri": "spotify:artist:a3", "profile": map[string]any{"name": "Third"}}}},
		}
		offset := getInt(vars, "offset")
		end := min(offset+getInt(vars, "limit"), len(artists))
		return jsonResponse(http.StatusOK, map[string]any{"data": map[string]any{"me": map[string]any{"libraryV3": map[string]any{
			"totalCount": len(artists), "items": artists[offset:end],
		}}}}), nil
	})
	client := newConnectClientForTests(transport)
	client.hashes.hashes["libraryV3"] = "hash"
	items, total, next, err := client.FollowedArtists(context.Background(), 1, "a1")
	if err != nil || total != 3 || len(items) != 1 || items[0].ID != "a2" || next != "a2" || requests != 2 {
		t.Fatalf("items=%#v total=%d next=%q requests=%d err=%v", items, total, next, requests, err)
	}
}

func TestConnectFollowedArtistCursorMissing(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]any{"data": map[string]any{"me": map[string]any{"libraryV3": map[string]any{
			"totalCount": 1, "items": []any{map[string]any{"item": map[string]any{"data": map[string]any{"uri": "spotify:artist:a1", "profile": map[string]any{"name": "First"}}}}},
		}}}}), nil
	})
	client := newConnectClientForTests(transport)
	client.hashes.hashes["libraryV3"] = "hash"
	if _, err := client.followedArtistOffset(context.Background(), "missing"); err == nil {
		t.Fatal("expected missing cursor error")
	}
}

func TestConnectLibraryTracksFallsBackWhenFetchLibraryTracksPayloadDrifts(t *testing.T) {
	webServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/me/tracks":
			_ = json.NewEncoder(w).Encode(libraryResponse{
				Items: []struct {
					Track trackItem `json:"track"`
					Album albumItem `json:"album"`
				}{
					{Track: trackItem{ID: "t1", URI: "spotify:track:t1", Name: "Song"}},
				},
				Total: 1,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(webServer.Close)

	webClient, err := NewClient(Options{
		TokenProvider: staticTokenProvider{},
		BaseURL:       webServer.URL,
		HTTPClient:    webServer.Client(),
	})
	if err != nil {
		t.Fatalf("web client: %v", err)
	}

	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("operationName") != "fetchLibraryTracks" {
			return textResponse(http.StatusNotFound, "missing"), nil
		}
		return jsonResponse(http.StatusOK, map[string]any{
			"data": map[string]any{"me": map[string]any{"library": map[string]any{}}},
		}), nil
	})
	client := newConnectClientForTests(transport)
	client.web = webClient
	client.hashes.hashes["fetchLibraryTracks"] = "hash"

	items, total, err := client.LibraryTracks(context.Background(), 10, 0)
	if err != nil || total != 1 || len(items) != 1 || items[0].ID != "t1" {
		t.Fatalf("library tracks fallback: items=%#v total=%d err=%v", items, total, err)
	}
}

func TestConnectInfoFallbackToWeb(t *testing.T) {
	infoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tracks/t1":
			_ = json.NewEncoder(w).Encode(trackItem{ID: "t1", URI: "spotify:track:t1", Name: "Song", Artists: []artistRef{{Name: "Artist"}}})
		case "/albums/a1":
			_ = json.NewEncoder(w).Encode(albumItem{ID: "a1", URI: "spotify:album:a1", Name: "Album", Artists: []artistRef{{Name: "Artist"}}})
		case "/artists/ar1":
			_ = json.NewEncoder(w).Encode(artistItem{ID: "ar1", URI: "spotify:artist:ar1", Name: "Artist"})
		case "/playlists/p1":
			_ = json.NewEncoder(w).Encode(playlistItem{ID: "p1", URI: "spotify:playlist:p1", Name: "Playlist"})
		case "/shows/s1":
			_ = json.NewEncoder(w).Encode(showItem{ID: "s1", URI: "spotify:show:s1", Name: "Show"})
		case "/episodes/e1":
			_ = json.NewEncoder(w).Encode(episodeItem{ID: "e1", URI: "spotify:episode:e1", Name: "Episode"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(infoServer.Close)

	webClient, err := NewClient(Options{
		TokenProvider: staticTokenProvider{},
		BaseURL:       infoServer.URL,
		HTTPClient:    infoServer.Client(),
	})
	if err != nil {
		t.Fatalf("web client: %v", err)
	}

	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return textResponse(http.StatusInternalServerError, "fail"), nil
	})
	client := newConnectClientForTests(transport)
	client.web = webClient
	client.hashes = &hashResolver{client: &http.Client{Transport: transport}, session: client.session, hashes: map[string]string{}}

	if item, err := client.GetTrack(context.Background(), "t1"); err != nil || item.ID != "t1" {
		t.Fatalf("track: %#v err=%v", item, err)
	}
	if item, err := client.GetAlbum(context.Background(), "a1"); err != nil || item.ID != "a1" {
		t.Fatalf("album: %#v err=%v", item, err)
	}
	if item, err := client.GetArtist(context.Background(), "ar1"); err != nil || item.ID != "ar1" {
		t.Fatalf("artist: %#v err=%v", item, err)
	}
	if item, err := client.GetPlaylist(context.Background(), "p1"); err != nil || item.ID != "p1" {
		t.Fatalf("playlist: %#v err=%v", item, err)
	}
	if item, err := client.GetShow(context.Background(), "s1"); err != nil || item.ID != "s1" {
		t.Fatalf("show: %#v err=%v", item, err)
	}
	if item, err := client.GetEpisode(context.Background(), "e1"); err != nil || item.ID != "e1" {
		t.Fatalf("episode: %#v err=%v", item, err)
	}
}

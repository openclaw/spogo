package spotify

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestNewConnectClient(t *testing.T) {
	if _, err := NewConnectClient(ConnectOptions{}); err == nil {
		t.Fatalf("expected error")
	}
	_, err := NewConnectClient(ConnectOptions{Source: cookieSourceStub{cookies: []*http.Cookie{{Name: "sp_dc", Value: "token"}}}})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
}

func TestConnectInfoOperations(t *testing.T) {
	payloads := map[string]map[string]any{
		"getTrack": {
			"data": map[string]any{"track": map[string]any{"uri": "spotify:track:t1", "name": "Song"}},
		},
		"getAlbum": {
			"data": map[string]any{"album": map[string]any{"uri": "spotify:album:a1", "name": "Album"}},
		},
		"queryArtistOverview": {
			"data": map[string]any{"artist": map[string]any{"uri": "spotify:artist:ar1", "name": "Artist"}},
		},
		"fetchPlaylist": {
			"data": map[string]any{"playlist": map[string]any{"uri": "spotify:playlist:p1", "name": "Playlist"}},
		},
		"queryPodcastEpisodes": {
			"data": map[string]any{"podcastUnionV2": map[string]any{"uri": "spotify:show:s1", "name": "Show"}},
		},
		"getEpisodeOrChapter": {
			"data": map[string]any{"episodeUnionV2": map[string]any{"uri": "spotify:episode:e1", "name": "Episode"}},
		},
	}
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		op := req.URL.Query().Get("operationName")
		if op == "getAlbum" || op == "queryPodcastEpisodes" || op == "getEpisodeOrChapter" {
			var variables map[string]any
			if err := json.Unmarshal([]byte(req.URL.Query().Get("variables")), &variables); err != nil {
				t.Fatalf("decode %s variables: %v", op, err)
			}
			switch op {
			case "getAlbum":
				if variables["uri"] != "spotify:album:a1" || variables["locale"] != "en-US" || variables["offset"] != float64(0) || variables["limit"] != float64(50) {
					t.Fatalf("unexpected album variables: %#v", variables)
				}
			case "queryPodcastEpisodes":
				if variables["uri"] != "spotify:show:s1" || variables["offset"] != float64(0) || variables["limit"] != float64(25) || variables["includeEpisodeContentRatingsV2"] != false {
					t.Fatalf("unexpected show variables: %#v", variables)
				}
			case "getEpisodeOrChapter":
				if variables["uri"] != "spotify:episode:e1" || variables["includeEpisodeContentRatingsV2"] != false {
					t.Fatalf("unexpected episode variables: %#v", variables)
				}
			}
		}
		payload, ok := payloads[op]
		if !ok {
			return textResponse(http.StatusNotFound, "missing"), nil
		}
		return jsonResponse(http.StatusOK, payload), nil
	})
	client := newConnectClientForTests(transport)
	client.language = "en-US"
	for op := range payloads {
		client.hashes.hashes[op] = "hash"
	}
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

func TestConnectUnsupported(t *testing.T) {
	client := &ConnectClient{}
	if _, _, err := client.LibraryTracks(context.Background(), 1, 0); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, err := client.LibraryAlbums(context.Background(), 1, 0); err == nil {
		t.Fatalf("expected error")
	}
	if err := client.LibraryModify(context.Background(), "", nil, ""); err == nil {
		t.Fatalf("expected error")
	}
	if err := client.FollowArtists(context.Background(), nil, ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, _, err := client.FollowedArtists(context.Background(), 1, ""); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, err := client.Playlists(context.Background(), 1, 0); err == nil {
		t.Fatalf("expected error")
	}
	if _, _, err := client.PlaylistTracks(context.Background(), "p1", 1, 0); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := client.CreatePlaylist(context.Background(), "name", false, false); err == nil {
		t.Fatalf("expected error")
	}
	if err := client.AddTracks(context.Background(), "p1", nil); err == nil {
		t.Fatalf("expected error")
	}
	if err := client.RemoveTracks(context.Background(), "p1", nil); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := client.GetUsersTopTracks(context.Background(), "long_term", 20, 0); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := client.GetRecentlyPlayed(context.Background(), 20, 0, 0); err == nil {
		t.Fatalf("expected error")
	}
}

func TestConnectUserReadsUseWebPlayerEndpoints(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Query().Get("operationName") {
		case "userTopContent":
			vars := req.URL.Query().Get("variables")
			if vars == "" {
				t.Fatalf("missing top content variables")
			}
			var variables map[string]any
			if err := json.Unmarshal([]byte(vars), &variables); err != nil {
				t.Fatalf("decode top content variables: %v", err)
			}
			topInput, ok := variables["topTracksInput"].(map[string]any)
			if !ok {
				t.Fatalf("missing topTracksInput")
			}
			if topInput["sortBy"] != "AFFINITY" || topInput["timeRange"] != "SHORT_TERM" {
				t.Fatalf("unexpected topTracksInput: %#v", topInput)
			}
			if req.URL.Query().Get("extensions") == "" {
				t.Fatalf("missing top content extensions")
			}
			return jsonResponse(http.StatusOK, map[string]any{
				"data": map[string]any{"me": map[string]any{"profile": map[string]any{
					"topTracks": map[string]any{
						"totalCount": 1,
						"items": []any{
							map[string]any{"data": map[string]any{
								"uri":          "spotify:track:t1",
								"name":         "Track",
								"albumOfTrack": map[string]any{"name": "Album"},
								"artists": map[string]any{"items": []any{
									map[string]any{"profile": map[string]any{"name": "Artist"}},
								}},
							}},
						},
					},
				}}},
			}), nil
		case "profileAttributes":
			return jsonResponse(http.StatusOK, map[string]any{
				"data": map[string]any{"me": map[string]any{"profile": map[string]any{
					"uri":      "spotify:user:user1",
					"username": "user1",
				}}},
			}), nil
		case "fetchEntitiesForRecentlyPlayed":
			return jsonResponse(http.StatusOK, map[string]any{
				"data": map[string]any{"lookup": []any{
					map[string]any{
						"_uri": "spotify:track:t1",
						"data": map[string]any{
							"uri":          "spotify:track:t1",
							"name":         "Track",
							"albumOfTrack": map[string]any{"name": "Album"},
							"artists": map[string]any{"items": []any{
								map[string]any{"profile": map[string]any{"name": "Artist"}},
							}},
						},
					},
				}},
			}), nil
		}
		if req.URL.Host == "spclient.wg.spotify.com" && req.URL.Path == "/recently-played/v3/user/user1/recently-played" {
			if got := req.URL.Query().Get("limit"); got != "50" {
				t.Fatalf("history page limit = %q, want 50", got)
			}
			if got := req.URL.Query().Get("offset"); got != "0" {
				t.Fatalf("history offset = %q, want 0", got)
			}
			return jsonResponse(http.StatusOK, recentlyPlayedContextsResponse{
				PlayContexts: []recentlyPlayedContext{
					{
						URI:                "spotify:playlist:p1",
						LastPlayedTime:     1705312800000,
						LastPlayedTrackURI: "spotify:track:t1",
					},
				},
			}), nil
		}
		return textResponse(http.StatusNotFound, "missing"), nil
	})
	client := newConnectClientForTests(transport)
	client.hashes.hashes["userTopContent"] = "hash"
	client.hashes.hashes["profileAttributes"] = "hash"
	client.hashes.hashes["fetchEntitiesForRecentlyPlayed"] = "hash"
	top, err := client.GetUsersTopTracks(context.Background(), "short_term", 5, 1)
	if err != nil {
		t.Fatalf("top tracks: %v", err)
	}
	if top.Total != 1 || top.Limit != 5 || top.Offset != 1 || len(top.Items) != 1 || top.Items[0].Album != "Album" {
		t.Fatalf("unexpected top tracks: %#v", top)
	}
	history, err := client.GetRecentlyPlayed(context.Background(), 6, 0, 1705312800001)
	if err != nil {
		t.Fatalf("recently played: %v", err)
	}
	if history.Limit != 6 || history.Cursors == nil || history.Cursors.Before != "1705312800000" || len(history.Items) != 1 {
		t.Fatalf("unexpected recently played: %#v", history)
	}
	if history.Items[0].Track.Name != "Track" || history.Items[0].PlayedAt != "2024-01-15T10:00:00.000Z" {
		t.Fatalf("unexpected recently played item: %#v", history.Items[0])
	}
}

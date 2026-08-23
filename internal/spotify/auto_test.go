package spotify

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/steipete/spogo/internal/cookies"
)

func TestAutoFallbackOnUnsupported(t *testing.T) {
	ctx := context.Background()
	calls := map[string]int{}
	connect := apiStub{
		calls: calls,
		searchFn: func(context.Context, string, string, int, int) (SearchResult, error) {
			return SearchResult{}, ErrUnsupported
		},
	}
	web := apiStub{
		calls: calls,
		searchFn: func(context.Context, string, string, int, int) (SearchResult, error) {
			return SearchResult{Type: "track"}, nil
		},
	}
	client := NewAutoClient(connect, web)
	res, err := client.Search(ctx, "track", "test", 10, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if res.Type != "track" {
		t.Fatalf("unexpected result: %#v", res)
	}
	if calls["Search"] != 2 {
		t.Fatalf("expected fallback search calls, got %d", calls["Search"])
	}
}

func TestAutoFallbackOnRateLimit(t *testing.T) {
	ctx := context.Background()
	calls := map[string]int{}
	connect := apiStub{
		calls: calls,
		playbackFn: func(context.Context) (PlaybackStatus, error) {
			return PlaybackStatus{}, APIError{Status: 429, Message: "rate limit"}
		},
	}
	web := apiStub{
		calls: calls,
		playbackFn: func(context.Context) (PlaybackStatus, error) {
			return PlaybackStatus{IsPlaying: true}, nil
		},
	}
	client := NewAutoClient(connect, web)
	status, err := client.Playback(ctx)
	if err != nil {
		t.Fatalf("playback: %v", err)
	}
	if !status.IsPlaying {
		t.Fatalf("expected playback")
	}
	if calls["Playback"] != 2 {
		t.Fatalf("expected fallback playback calls, got %d", calls["Playback"])
	}
}

func TestAutoLibraryFallbacks(t *testing.T) {
	ctx := context.Background()
	calls := map[string]int{}
	connect := apiStub{
		calls: calls,
		libraryTracksFn: func(context.Context, int, int) ([]Item, int, error) {
			return nil, 0, ErrUnsupported
		},
		followedArtistsFn: func(context.Context, int, string) ([]Item, int, string, error) {
			return nil, 0, "", ErrUnsupported
		},
		libraryModifyFn: func(context.Context, string, []string, string) error {
			return ErrUnsupported
		},
	}
	web := apiStub{
		calls: calls,
		libraryTracksFn: func(context.Context, int, int) ([]Item, int, error) {
			return []Item{{ID: "1"}}, 1, nil
		},
		followedArtistsFn: func(context.Context, int, string) ([]Item, int, string, error) {
			return []Item{{ID: "2"}}, 1, "next", nil
		},
		libraryModifyFn: func(context.Context, string, []string, string) error {
			return nil
		},
	}
	client := NewAutoClient(connect, web)
	items, total, err := client.LibraryTracks(ctx, 10, 0)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("library tracks: %v %#v %d", err, items, total)
	}
	artists, count, after, err := client.FollowedArtists(ctx, 10, "")
	if err != nil || count != 1 || after != "next" || len(artists) != 1 {
		t.Fatalf("followed artists: %v %#v %d %s", err, artists, count, after)
	}
	if err := client.LibraryModify(ctx, "me/tracks", []string{"1"}, "put"); err != nil {
		t.Fatalf("library modify: %v", err)
	}
	if calls["LibraryTracks"] != 2 || calls["FollowedArtists"] != 2 || calls["LibraryModify"] != 2 {
		t.Fatalf("expected fallback calls: %#v", calls)
	}
}

func TestAutoNoFallbackOnGenericError(t *testing.T) {
	ctx := context.Background()
	calls := map[string]int{}
	connect := apiStub{
		calls: calls,
		playbackFn: func(context.Context) (PlaybackStatus, error) {
			return PlaybackStatus{}, errors.New("boom")
		},
	}
	web := apiStub{
		calls: calls,
		playbackFn: func(context.Context) (PlaybackStatus, error) {
			return PlaybackStatus{IsPlaying: true}, nil
		},
	}
	client := NewAutoClient(connect, web)
	if _, err := client.Playback(ctx); err == nil {
		t.Fatalf("expected error")
	}
	if calls["Playback"] != 1 {
		t.Fatalf("expected no fallback, got %d", calls["Playback"])
	}
}

func TestAutoPlaybackFallsBackToLocalAfterBothRemoteEnginesFail(t *testing.T) {
	connectCalls := map[string]int{}
	webCalls := map[string]int{}
	localCalls := map[string]int{}
	connect := apiStub{calls: connectCalls, playbackFn: func(context.Context) (PlaybackStatus, error) {
		return PlaybackStatus{}, cookies.ErrNoCookies
	}}
	web := apiStub{calls: webCalls, playbackFn: func(context.Context) (PlaybackStatus, error) {
		return PlaybackStatus{}, cookies.ErrNoCookies
	}}
	local := apiStub{calls: localCalls, playbackFn: func(context.Context) (PlaybackStatus, error) {
		return PlaybackStatus{IsPlaying: true, Device: Device{Name: "Local Spotify"}}, nil
	}}
	client := NewAutoClient(connect, web, local)
	playback, err := client.Playback(context.Background())
	if err != nil || !playback.IsPlaying || playback.Device.Name != "Local Spotify" {
		t.Fatalf("playback=%#v err=%v", playback, err)
	}
	if connectCalls["Playback"] != 1 || webCalls["Playback"] != 1 || localCalls["Playback"] != 1 {
		t.Fatalf("fallback order connect=%#v web=%#v local=%#v", connectCalls, webCalls, localCalls)
	}
}

func TestAutoPlaybackControlFallsBackToLocalAfterRateLimits(t *testing.T) {
	connectCalls := map[string]int{}
	webCalls := map[string]int{}
	localCalls := map[string]int{}
	remoteError := APIError{Status: 429, RetryAfter: 24 * time.Hour}
	connect := apiStub{calls: connectCalls, pauseFn: func(context.Context) error { return remoteError }}
	web := apiStub{calls: webCalls, pauseFn: func(context.Context) error { return remoteError }}
	local := apiStub{calls: localCalls, pauseFn: func(context.Context) error { return nil }}
	if err := NewAutoClient(connect, web, local).Pause(context.Background()); err != nil {
		t.Fatal(err)
	}
	if connectCalls["Pause"] != 1 || webCalls["Pause"] != 1 || localCalls["Pause"] != 1 {
		t.Fatalf("fallback order connect=%#v web=%#v local=%#v", connectCalls, webCalls, localCalls)
	}
}

func TestAutoUsesLocalOnlyForPlaybackCommands(t *testing.T) {
	localCalls := map[string]int{}
	connect := apiStub{
		searchFn: func(context.Context, string, string, int, int) (SearchResult, error) {
			return SearchResult{}, ErrUnsupported
		},
		libraryTracksFn: func(context.Context, int, int) ([]Item, int, error) {
			return nil, 0, ErrUnsupported
		},
	}
	web := apiStub{
		searchFn: func(context.Context, string, string, int, int) (SearchResult, error) {
			return SearchResult{}, errors.New("web unavailable")
		},
		libraryTracksFn: func(context.Context, int, int) ([]Item, int, error) {
			return nil, 0, errors.New("web unavailable")
		},
	}
	local := apiStub{calls: localCalls}
	client := NewAutoClient(connect, web, local)
	if _, err := client.Search(context.Background(), "track", "weezer", 1, 0); err == nil {
		t.Fatal("expected remote search failure")
	}
	if _, _, err := client.LibraryTracks(context.Background(), 1, 0); err == nil {
		t.Fatal("expected remote library failure")
	}
	if len(localCalls) != 0 {
		t.Fatalf("unsupported commands reached local Spotify: %#v", localCalls)
	}
}

func TestAutoSkipsLocalWhenRemotePlaybackSucceeds(t *testing.T) {
	localCalls := map[string]int{}
	client := NewAutoClient(apiStub{playbackFn: func(context.Context) (PlaybackStatus, error) {
		return PlaybackStatus{IsPlaying: true}, nil
	}}, apiStub{}, apiStub{calls: localCalls})
	if _, err := client.Playback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(localCalls) != 0 {
		t.Fatalf("local Spotify used before remote failure: %#v", localCalls)
	}
}

func TestAutoPreservesRateLimitHintWhenFallbackFails(t *testing.T) {
	primary := APIError{Status: 429, Message: "rate limit", RetryAfter: 24 * time.Hour}
	connect := apiStub{searchFn: func(context.Context, string, string, int, int) (SearchResult, error) {
		return SearchResult{}, primary
	}}
	web := apiStub{searchFn: func(context.Context, string, string, int, int) (SearchResult, error) {
		return SearchResult{}, errors.New("secondary unavailable")
	}}
	_, err := NewAutoClient(connect, web).Search(context.Background(), "track", "weezer", 1, 0)
	var apiErr APIError
	if !errors.As(err, &apiErr) || apiErr.RetryAfter != 24*time.Hour {
		t.Fatalf("expected actionable retry hint, got %v", err)
	}
}

func TestAutoArtistTopTracksFallback(t *testing.T) {
	ctx := context.Background()
	calls := map[string]int{}
	connect := apiStub{
		calls: calls,
		artistTopTracksFn: func(context.Context, string, int) ([]Item, error) {
			return nil, ErrUnsupported
		},
	}
	web := apiStub{
		calls: calls,
		artistTopTracksFn: func(context.Context, string, int) ([]Item, error) {
			return []Item{{URI: "spotify:track:1"}}, nil
		},
	}
	client := NewAutoClient(connect, web)
	auto, ok := client.(artistTopTracksAPI)
	if !ok {
		t.Fatalf("expected artist top tracks support")
	}
	items, err := auto.ArtistTopTracks(ctx, "abc", 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("artist top tracks: %v %#v", err, items)
	}
	if calls["ArtistTopTracks"] != 2 {
		t.Fatalf("expected fallback calls, got %d", calls["ArtistTopTracks"])
	}
}

func TestAutoPassThrough(t *testing.T) {
	ctx := context.Background()
	connectCalls := map[string]int{}
	webCalls := map[string]int{}
	connect := apiStub{calls: connectCalls}
	web := apiStub{calls: webCalls}
	client := NewAutoClient(connect, web)

	_, _ = client.Search(ctx, "track", "test", 1, 0)
	_, _ = client.GetTrack(ctx, "1")
	_, _ = client.GetAlbum(ctx, "1")
	_, _ = client.GetArtist(ctx, "1")
	_, _ = client.GetPlaylist(ctx, "1")
	_, _ = client.GetShow(ctx, "1")
	_, _ = client.GetEpisode(ctx, "1")
	_, _ = client.Playback(ctx)
	_ = client.Play(ctx, "spotify:track:1")
	_ = client.Pause(ctx)
	_ = client.Next(ctx)
	_ = client.Previous(ctx)
	_ = client.Seek(ctx, 10)
	_ = client.Volume(ctx, 50)
	_ = client.Shuffle(ctx, true)
	_ = client.Repeat(ctx, "off")
	_, _ = client.Devices(ctx)
	_ = client.Transfer(ctx, "device")
	_ = client.QueueAdd(ctx, "spotify:track:1")
	_, _ = client.Queue(ctx)
	_, _, _ = client.LibraryTracks(ctx, 1, 0)
	_, _, _ = client.LibraryAlbums(ctx, 1, 0)
	_ = client.LibraryModify(ctx, "me/tracks", []string{"1"}, "put")
	_ = client.FollowArtists(ctx, []string{"1"}, "put")
	_, _, _, _ = client.FollowedArtists(ctx, 1, "")
	_, _, _ = client.Playlists(ctx, 1, 0)
	_, _, _ = client.PlaylistTracks(ctx, "1", 1, 0)
	_, _ = client.CreatePlaylist(ctx, "name", false, false)
	_ = client.AddTracks(ctx, "1", []string{"spotify:track:1"})
	_ = client.RemoveTracks(ctx, "1", []string{"spotify:track:1"})
	_, _ = client.GetUsersTopTracks(ctx, "long_term", 20, 0)
	_, _ = client.GetRecentlyPlayed(ctx, 20, 0, 0)

	if len(webCalls) != 0 {
		t.Fatalf("expected no web calls, got %#v", webCalls)
	}
	if connectCalls["Search"] == 0 || connectCalls["Play"] == 0 || connectCalls["RemoveTracks"] == 0 || connectCalls["GetUsersTopTracks"] == 0 || connectCalls["GetRecentlyPlayed"] == 0 {
		t.Fatalf("expected connect calls, got %#v", connectCalls)
	}
}

func TestAutoUserReadsFallback(t *testing.T) {
	ctx := context.Background()
	calls := map[string]int{}
	connect := apiStub{
		calls: calls,
		topTracksFn: func(context.Context, string, int, int) (TopTracksResult, error) {
			return TopTracksResult{}, ErrUnsupported
		},
		recentlyPlayedFn: func(context.Context, int, int64, int64) (RecentlyPlayedResult, error) {
			return RecentlyPlayedResult{}, ErrUnsupported
		},
	}
	web := apiStub{
		calls: calls,
		topTracksFn: func(context.Context, string, int, int) (TopTracksResult, error) {
			return TopTracksResult{Items: []Item{{ID: "t1"}}}, nil
		},
		recentlyPlayedFn: func(context.Context, int, int64, int64) (RecentlyPlayedResult, error) {
			return RecentlyPlayedResult{Items: []RecentlyPlayedItem{{Track: Item{ID: "t1"}}}}, nil
		},
	}
	client := NewAutoClient(connect, web)
	top, err := client.GetUsersTopTracks(ctx, "medium_term", 5, 1)
	if err != nil {
		t.Fatalf("top tracks: %v", err)
	}
	if len(top.Items) != 1 {
		t.Fatalf("unexpected top tracks result: %#v", top)
	}
	history, err := client.GetRecentlyPlayed(ctx, 5, 0, 123)
	if err != nil {
		t.Fatalf("recently played: %v", err)
	}
	if len(history.Items) != 1 {
		t.Fatalf("unexpected recently played result: %#v", history)
	}
	if calls["GetUsersTopTracks"] != 2 || calls["GetRecentlyPlayed"] != 2 {
		t.Fatalf("expected fallback calls, got %#v", calls)
	}
}

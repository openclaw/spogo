package spotify

import (
	"context"
	"testing"
)

type apiStub struct {
	calls             map[string]int
	playbackFn        func(context.Context) (PlaybackStatus, error)
	pauseFn           func(context.Context) error
	searchFn          func(context.Context, string, string, int, int) (SearchResult, error)
	devicesFn         func(context.Context) ([]Device, error)
	libraryTracksFn   func(context.Context, int, int) ([]Item, int, error)
	libraryModifyFn   func(context.Context, string, []string, string) error
	followedArtistsFn func(context.Context, int, string) ([]Item, int, string, error)
	artistTopTracksFn func(context.Context, string, int) ([]Item, error)
	addTracksFn       func(context.Context, string, []string) error
	removeTracksFn    func(context.Context, string, []string) error
	topTracksFn       func(context.Context, string, int, int) (TopTracksResult, error)
	recentlyPlayedFn  func(context.Context, int, int64, int64) (RecentlyPlayedResult, error)
}

func (a apiStub) Search(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	a.note("Search")
	if a.searchFn != nil {
		return a.searchFn(ctx, kind, query, limit, offset)
	}
	return SearchResult{}, nil
}

func (a apiStub) GetTrack(context.Context, string) (Item, error) {
	a.note("GetTrack")
	return Item{}, nil
}

func (a apiStub) GetAlbum(context.Context, string) (Item, error) {
	a.note("GetAlbum")
	return Item{}, nil
}

func (a apiStub) GetArtist(context.Context, string) (Item, error) {
	a.note("GetArtist")
	return Item{}, nil
}

func (a apiStub) GetPlaylist(context.Context, string) (Item, error) {
	a.note("GetPlaylist")
	return Item{}, nil
}

func (a apiStub) GetShow(context.Context, string) (Item, error) {
	a.note("GetShow")
	return Item{}, nil
}

func (a apiStub) GetEpisode(context.Context, string) (Item, error) {
	a.note("GetEpisode")
	return Item{}, nil
}

func (a apiStub) Playback(ctx context.Context) (PlaybackStatus, error) {
	a.note("Playback")
	if a.playbackFn != nil {
		return a.playbackFn(ctx)
	}
	return PlaybackStatus{}, nil
}

func (a apiStub) Play(context.Context, string) error {
	a.note("Play")
	return nil
}

func (a apiStub) PlayTracks(context.Context, []string) error {
	a.note("PlayTracks")
	return nil
}

func (a apiStub) Pause(ctx context.Context) error {
	a.note("Pause")
	if a.pauseFn != nil {
		return a.pauseFn(ctx)
	}
	return nil
}

func (a apiStub) Next(context.Context) error {
	a.note("Next")
	return nil
}

func (a apiStub) Previous(context.Context) error {
	a.note("Previous")
	return nil
}

func (a apiStub) Seek(context.Context, int) error {
	a.note("Seek")
	return nil
}

func (a apiStub) Volume(context.Context, int) error {
	a.note("Volume")
	return nil
}

func (a apiStub) Shuffle(context.Context, bool) error {
	a.note("Shuffle")
	return nil
}

func (a apiStub) Repeat(context.Context, string) error {
	a.note("Repeat")
	return nil
}

func (a apiStub) Devices(ctx context.Context) ([]Device, error) {
	a.note("Devices")
	if a.devicesFn != nil {
		return a.devicesFn(ctx)
	}
	return nil, nil
}

func (a apiStub) ArtistTopTracks(ctx context.Context, id string, limit int) ([]Item, error) {
	a.note("ArtistTopTracks")
	if a.artistTopTracksFn != nil {
		return a.artistTopTracksFn(ctx, id, limit)
	}
	return nil, nil
}

func (a apiStub) Transfer(context.Context, string) error {
	a.note("Transfer")
	return nil
}

func (a apiStub) QueueAdd(context.Context, string) error {
	a.note("QueueAdd")
	return nil
}

func (a apiStub) Queue(context.Context) (Queue, error) {
	a.note("Queue")
	return Queue{}, nil
}

func (a apiStub) LibraryTracks(ctx context.Context, limit, offset int) ([]Item, int, error) {
	a.note("LibraryTracks")
	if a.libraryTracksFn != nil {
		return a.libraryTracksFn(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (a apiStub) LibraryAlbums(context.Context, int, int) ([]Item, int, error) {
	a.note("LibraryAlbums")
	return nil, 0, nil
}

func (a apiStub) LibraryModify(ctx context.Context, path string, ids []string, method string) error {
	a.note("LibraryModify")
	if a.libraryModifyFn != nil {
		return a.libraryModifyFn(ctx, path, ids, method)
	}
	return nil
}

func (a apiStub) FollowArtists(context.Context, []string, string) error {
	a.note("FollowArtists")
	return nil
}

func (a apiStub) FollowedArtists(ctx context.Context, limit int, after string) ([]Item, int, string, error) {
	a.note("FollowedArtists")
	if a.followedArtistsFn != nil {
		return a.followedArtistsFn(ctx, limit, after)
	}
	return nil, 0, "", nil
}

func (a apiStub) Playlists(context.Context, int, int) ([]Item, int, error) {
	a.note("Playlists")
	return nil, 0, nil
}

func (a apiStub) PlaylistTracks(context.Context, string, int, int) ([]Item, int, error) {
	a.note("PlaylistTracks")
	return nil, 0, nil
}

func (a apiStub) CreatePlaylist(context.Context, string, bool, bool) (Item, error) {
	a.note("CreatePlaylist")
	return Item{}, nil
}

func (a apiStub) AddTracks(ctx context.Context, playlistID string, uris []string) error {
	a.note("AddTracks")
	if a.addTracksFn != nil {
		return a.addTracksFn(ctx, playlistID, uris)
	}
	return nil
}

func (a apiStub) RemoveTracks(ctx context.Context, playlistID string, uris []string) error {
	a.note("RemoveTracks")
	if a.removeTracksFn != nil {
		return a.removeTracksFn(ctx, playlistID, uris)
	}
	return nil
}

func (a apiStub) GetUsersTopTracks(ctx context.Context, timeRange string, limit, offset int) (TopTracksResult, error) {
	a.note("GetUsersTopTracks")
	if a.topTracksFn != nil {
		return a.topTracksFn(ctx, timeRange, limit, offset)
	}
	return TopTracksResult{}, nil
}

func (a apiStub) GetRecentlyPlayed(ctx context.Context, limit int, after, before int64) (RecentlyPlayedResult, error) {
	a.note("GetRecentlyPlayed")
	if a.recentlyPlayedFn != nil {
		return a.recentlyPlayedFn(ctx, limit, after, before)
	}
	return RecentlyPlayedResult{}, nil
}

func (a apiStub) note(name string) {
	if a.calls == nil {
		return
	}
	a.calls[name]++
}

func TestFallbackPlaybackOnRateLimit(t *testing.T) {
	ctx := context.Background()
	webCalls := 0
	connectCalls := 0
	web := apiStub{
		playbackFn: func(context.Context) (PlaybackStatus, error) {
			webCalls++
			return PlaybackStatus{}, APIError{Status: 429, Message: "rate limit"}
		},
	}
	connect := apiStub{
		playbackFn: func(context.Context) (PlaybackStatus, error) {
			connectCalls++
			return PlaybackStatus{IsPlaying: true}, nil
		},
	}
	client := NewPlaybackFallbackClient(web, connect)
	status, err := client.Playback(ctx)
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if !status.IsPlaying {
		t.Fatalf("expected playback from fallback")
	}
	if webCalls != 1 || connectCalls != 1 {
		t.Fatalf("unexpected call counts web=%d connect=%d", webCalls, connectCalls)
	}
}

func TestFallbackArtistTopTracksOnRateLimit(t *testing.T) {
	ctx := context.Background()
	web := apiStub{
		artistTopTracksFn: func(context.Context, string, int) ([]Item, error) {
			return nil, APIError{Status: 429, Message: "rate limit"}
		},
	}
	connect := apiStub{
		artistTopTracksFn: func(context.Context, string, int) ([]Item, error) {
			return []Item{{URI: "spotify:track:1"}}, nil
		},
	}
	client := NewPlaybackFallbackClient(web, connect)
	fallback, ok := client.(artistTopTracksAPI)
	if !ok {
		t.Fatalf("expected artist top tracks support")
	}
	items, err := fallback.ArtistTopTracks(ctx, "abc", 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("artist top tracks: %v %#v", err, items)
	}
}

func TestFallbackSkipsNonRateLimit(t *testing.T) {
	ctx := context.Background()
	webCalls := 0
	connectCalls := 0
	web := apiStub{
		playbackFn: func(context.Context) (PlaybackStatus, error) {
			webCalls++
			return PlaybackStatus{}, APIError{Status: 500, Message: "boom"}
		},
	}
	connect := apiStub{
		playbackFn: func(context.Context) (PlaybackStatus, error) {
			connectCalls++
			return PlaybackStatus{IsPlaying: true}, nil
		},
	}
	client := NewPlaybackFallbackClient(web, connect)
	if _, err := client.Playback(ctx); err == nil {
		t.Fatalf("expected error")
	}
	if webCalls != 1 || connectCalls != 0 {
		t.Fatalf("unexpected call counts web=%d connect=%d", webCalls, connectCalls)
	}
}

func TestFallbackSearchOnRateLimit(t *testing.T) {
	ctx := context.Background()
	webCalls := 0
	connectCalls := 0
	web := apiStub{
		searchFn: func(context.Context, string, string, int, int) (SearchResult, error) {
			webCalls++
			return SearchResult{}, APIError{Status: 429, Message: "rate limit"}
		},
	}
	connect := apiStub{
		searchFn: func(context.Context, string, string, int, int) (SearchResult, error) {
			connectCalls++
			return SearchResult{Type: "track"}, nil
		},
	}
	client := NewPlaybackFallbackClient(web, connect)
	if _, err := client.Search(ctx, "track", "test", 1, 0); err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if webCalls != 1 || connectCalls != 1 {
		t.Fatalf("unexpected call counts web=%d connect=%d", webCalls, connectCalls)
	}
}

func TestFallbackPauseOnRateLimit(t *testing.T) {
	ctx := context.Background()
	webCalls := 0
	connectCalls := 0
	web := apiStub{
		pauseFn: func(context.Context) error {
			webCalls++
			return APIError{Status: 429, Message: "rate limit"}
		},
	}
	connect := apiStub{
		pauseFn: func(context.Context) error {
			connectCalls++
			return nil
		},
	}
	client := NewPlaybackFallbackClient(web, connect)
	if err := client.Pause(ctx); err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}
	if webCalls != 1 || connectCalls != 1 {
		t.Fatalf("unexpected call counts web=%d connect=%d", webCalls, connectCalls)
	}
}

func TestFallbackPlaylistWritesOnRateLimit(t *testing.T) {
	ctx := context.Background()
	calls := map[string]int{}
	web := apiStub{
		calls: calls,
		addTracksFn: func(context.Context, string, []string) error {
			return APIError{Status: 429, Message: "rate limit"}
		},
		removeTracksFn: func(context.Context, string, []string) error {
			return APIError{Status: 429, Message: "rate limit"}
		},
	}
	connect := apiStub{calls: calls}
	client := NewPlaybackFallbackClient(web, connect)
	if err := client.AddTracks(ctx, "p1", []string{"spotify:track:t1"}); err != nil {
		t.Fatalf("add tracks: %v", err)
	}
	if err := client.RemoveTracks(ctx, "p1", []string{"spotify:track:t1"}); err != nil {
		t.Fatalf("remove tracks: %v", err)
	}
	if calls["AddTracks"] != 2 || calls["RemoveTracks"] != 2 {
		t.Fatalf("unexpected calls: %#v", calls)
	}
}

func TestFallbackDelegatesToWeb(t *testing.T) {
	ctx := context.Background()
	calls := map[string]int{}
	web := apiStub{calls: calls}
	client := NewPlaybackFallbackClient(web, apiStub{})

	if _, err := client.GetTrack(ctx, "t1"); err != nil {
		t.Fatalf("get track: %v", err)
	}
	if _, err := client.GetAlbum(ctx, "a1"); err != nil {
		t.Fatalf("get album: %v", err)
	}
	if _, err := client.GetArtist(ctx, "ar1"); err != nil {
		t.Fatalf("get artist: %v", err)
	}
	if _, err := client.GetPlaylist(ctx, "p1"); err != nil {
		t.Fatalf("get playlist: %v", err)
	}
	if _, err := client.GetShow(ctx, "s1"); err != nil {
		t.Fatalf("get show: %v", err)
	}
	if _, err := client.GetEpisode(ctx, "e1"); err != nil {
		t.Fatalf("get episode: %v", err)
	}
	if err := client.Play(ctx, "spotify:track:t1"); err != nil {
		t.Fatalf("play: %v", err)
	}
	if err := client.Next(ctx); err != nil {
		t.Fatalf("next: %v", err)
	}
	if err := client.Previous(ctx); err != nil {
		t.Fatalf("previous: %v", err)
	}
	if err := client.Seek(ctx, 1000); err != nil {
		t.Fatalf("seek: %v", err)
	}
	if err := client.Volume(ctx, 10); err != nil {
		t.Fatalf("volume: %v", err)
	}
	if err := client.Shuffle(ctx, true); err != nil {
		t.Fatalf("shuffle: %v", err)
	}
	if err := client.Repeat(ctx, "off"); err != nil {
		t.Fatalf("repeat: %v", err)
	}
	if _, err := client.Devices(ctx); err != nil {
		t.Fatalf("devices: %v", err)
	}
	if err := client.Transfer(ctx, "d1"); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if err := client.QueueAdd(ctx, "spotify:track:t1"); err != nil {
		t.Fatalf("queue add: %v", err)
	}
	if _, err := client.Queue(ctx); err != nil {
		t.Fatalf("queue: %v", err)
	}
	if _, _, err := client.LibraryTracks(ctx, 1, 0); err != nil {
		t.Fatalf("library tracks: %v", err)
	}
	if _, _, err := client.LibraryAlbums(ctx, 1, 0); err != nil {
		t.Fatalf("library albums: %v", err)
	}
	if err := client.LibraryModify(ctx, "/me/tracks", []string{"t1"}, "PUT"); err != nil {
		t.Fatalf("library modify: %v", err)
	}
	if err := client.FollowArtists(ctx, []string{"ar1"}, "PUT"); err != nil {
		t.Fatalf("follow artists: %v", err)
	}
	if _, _, _, err := client.FollowedArtists(ctx, 1, ""); err != nil {
		t.Fatalf("followed artists: %v", err)
	}
	if _, _, err := client.Playlists(ctx, 1, 0); err != nil {
		t.Fatalf("playlists: %v", err)
	}
	if _, _, err := client.PlaylistTracks(ctx, "p1", 1, 0); err != nil {
		t.Fatalf("playlist tracks: %v", err)
	}
	if _, err := client.CreatePlaylist(ctx, "Name", true, false); err != nil {
		t.Fatalf("create playlist: %v", err)
	}
	if err := client.AddTracks(ctx, "p1", []string{"spotify:track:t1"}); err != nil {
		t.Fatalf("add tracks: %v", err)
	}
	if err := client.RemoveTracks(ctx, "p1", []string{"spotify:track:t1"}); err != nil {
		t.Fatalf("remove tracks: %v", err)
	}
	if _, err := client.GetUsersTopTracks(ctx, "long_term", 20, 0); err != nil {
		t.Fatalf("top tracks: %v", err)
	}
	if _, err := client.GetRecentlyPlayed(ctx, 20, 0, 0); err != nil {
		t.Fatalf("recently played: %v", err)
	}

	if calls["GetTrack"] == 0 || calls["Queue"] == 0 || calls["RemoveTracks"] == 0 || calls["GetUsersTopTracks"] == 0 || calls["GetRecentlyPlayed"] == 0 {
		t.Fatalf("expected web calls to be recorded")
	}
}

func TestFallbackUserReadsOnRateLimit(t *testing.T) {
	ctx := context.Background()
	calls := map[string]int{}
	web := apiStub{
		calls: calls,
		topTracksFn: func(context.Context, string, int, int) (TopTracksResult, error) {
			return TopTracksResult{}, APIError{Status: 429, Message: "rate limit"}
		},
		recentlyPlayedFn: func(context.Context, int, int64, int64) (RecentlyPlayedResult, error) {
			return RecentlyPlayedResult{}, APIError{Status: 429, Message: "rate limit"}
		},
	}
	connect := apiStub{
		calls: calls,
		topTracksFn: func(context.Context, string, int, int) (TopTracksResult, error) {
			return TopTracksResult{Total: 1, Items: []Item{{ID: "t1"}}}, nil
		},
		recentlyPlayedFn: func(context.Context, int, int64, int64) (RecentlyPlayedResult, error) {
			return RecentlyPlayedResult{Items: []RecentlyPlayedItem{{Track: Item{ID: "t1"}}}}, nil
		},
	}
	client := NewPlaybackFallbackClient(web, connect)
	top, err := client.GetUsersTopTracks(ctx, "short_term", 10, 2)
	if err != nil {
		t.Fatalf("top tracks: %v", err)
	}
	if top.Total != 1 || len(top.Items) != 1 {
		t.Fatalf("unexpected top tracks result: %#v", top)
	}
	history, err := client.GetRecentlyPlayed(ctx, 10, 0, 123)
	if err != nil {
		t.Fatalf("recently played: %v", err)
	}
	if len(history.Items) != 1 {
		t.Fatalf("unexpected recently played result: %#v", history)
	}
	if calls["GetUsersTopTracks"] != 2 || calls["GetRecentlyPlayed"] != 2 {
		t.Fatalf("expected fallback calls: %#v", calls)
	}
}

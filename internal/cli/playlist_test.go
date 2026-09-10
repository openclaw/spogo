package cli

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestPlaylistAddCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	called := false
	mock := &testutil.SpotifyMock{
		AddTracksFn: func(ctx context.Context, playlistID string, uris []string) error {
			called = true
			if playlistID != "p1" {
				t.Fatalf("playlist id %s", playlistID)
			}
			if len(uris) != 1 || uris[0] != "spotify:track:t1" {
				t.Fatalf("uris: %#v", uris)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistAddCmd{Playlist: "spotify:playlist:p1", Tracks: []string{"spotify:track:t1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatalf("expected call")
	}
}

func TestPlaylistAddCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		AddTracksFn: func(ctx context.Context, playlistID string, uris []string) error {
			return errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistAddCmd{Playlist: "spotify:playlist:p1", Tracks: []string{"spotify:track:t1"}}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlaylistCreateCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		CreatePlaylistFn: func(ctx context.Context, name string, public, collaborative bool) (spotify.Item, error) {
			return spotify.Item{ID: "p1", Name: name, Type: "playlist"}, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistCreateCmd{Name: "Road Trip"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlaylistTracksCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		PlaylistTracksFn: func(ctx context.Context, id string, limit, offset int) ([]spotify.Item, int, error) {
			return []spotify.Item{{ID: "t1", Name: "Track", Type: "track"}}, 1, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistTracksCmd{Playlist: "spotify:playlist:p1", Limit: 1}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlaylistRemoveCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		RemoveTracksFn: func(ctx context.Context, playlistID string, uris []string) error {
			if playlistID != "p1" {
				t.Fatalf("playlist %s", playlistID)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistRemoveCmd{Playlist: "spotify:playlist:p1", Tracks: []string{"spotify:track:t1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestPlaylistFollowCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	var gotID string
	var gotPublic bool
	mock := &testutil.SpotifyMock{
		FollowPlaylistFn: func(ctx context.Context, id string, public bool) error {
			gotID = id
			gotPublic = public
			return nil
		},
		GetPlaylistFn: func(ctx context.Context, id string) (spotify.Item, error) {
			return spotify.Item{ID: id, Name: "Road Trip", Type: "playlist"}, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistFollowCmd{Playlist: "https://open.spotify.com/playlist/p1", Public: true}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotID != "p1" || !gotPublic {
		t.Fatalf("got id=%s public=%v", gotID, gotPublic)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlaylistFollowCmdDefaultPublicFalse(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	var gotPublic bool
	called := false
	mock := &testutil.SpotifyMock{
		FollowPlaylistFn: func(ctx context.Context, id string, public bool) error {
			called = true
			gotPublic = public
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistFollowCmd{Playlist: "spotify:playlist:p1"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called || gotPublic {
		t.Fatalf("expected explicit public=false, got called=%v public=%v", called, gotPublic)
	}
}

func TestPlaylistFollowCmdNameLookupFailureFallsBackToID(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		FollowPlaylistFn: func(ctx context.Context, id string, public bool) error {
			return nil
		},
		GetPlaylistFn: func(ctx context.Context, id string) (spotify.Item, error) {
			return spotify.Item{}, errors.New("lookup failed")
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistFollowCmd{Playlist: "spotify:playlist:p1"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run should not fail when name lookup fails: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlaylistFollowCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		FollowPlaylistFn: func(ctx context.Context, id string, public bool) error {
			return errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistFollowCmd{Playlist: "spotify:playlist:p1"}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlaylistUnfollowCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	var gotID string
	mock := &testutil.SpotifyMock{
		UnfollowPlaylistFn: func(ctx context.Context, id string) error {
			gotID = id
			return nil
		},
		GetPlaylistFn: func(ctx context.Context, id string) (spotify.Item, error) {
			return spotify.Item{ID: id, Name: "Road Trip", Type: "playlist"}, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistUnfollowCmd{Playlist: "spotify:playlist:p1"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if gotID != "p1" {
		t.Fatalf("got id=%s", gotID)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlaylistUnfollowCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		UnfollowPlaylistFn: func(ctx context.Context, id string) error {
			return errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistUnfollowCmd{Playlist: "spotify:playlist:p1"}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlaylistFollowingCmdTrue(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		IsFollowingPlaylistFn: func(ctx context.Context, id string) (bool, error) {
			return true, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistFollowingCmd{Playlist: "spotify:playlist:p1"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlaylistFollowingCmdFalse(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		IsFollowingPlaylistFn: func(ctx context.Context, id string) (bool, error) {
			return false, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistFollowingCmd{Playlist: "spotify:playlist:p1"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlaylistFollowingCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		IsFollowingPlaylistFn: func(ctx context.Context, id string) (bool, error) {
			return false, errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlaylistFollowingCmd{Playlist: "spotify:playlist:p1"}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

type playlistTokenProvider struct{}

func (playlistTokenProvider) Token(context.Context) (spotify.Token, error) {
	return spotify.Token{AccessToken: "test-token"}, nil
}

func TestPlaylistLibraryEndpoints(t *testing.T) {
	following := false
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/playlists/p1" {
			http.Error(w, "unavailable", http.StatusNotFound)
			return
		}
		requests++
		if r.URL.Query().Encode() != "uris=spotify%3Aplaylist%3Ap1" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/me/library":
			following = true
		case r.Method == http.MethodDelete && r.URL.Path == "/me/library":
			following = false
		case r.Method == http.MethodGet && r.URL.Path == "/me/library/contains":
			if following {
				_, _ = w.Write([]byte(`[true]`))
			} else {
				_, _ = w.Write([]byte(`[false]`))
			}
			return
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	client, err := spotify.NewClient(spotify.Options{TokenProvider: playlistTokenProvider{}, BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(client)
	for i, want := range []string{"false\n", "true\n", "false\n"} {
		if i == 1 {
			if err := (&PlaylistFollowCmd{Playlist: "https://open.spotify.com/playlist/p1", Public: true}).Run(ctx); err != nil {
				t.Fatal(err)
			}
		}
		if i == 2 {
			if err := (&PlaylistUnfollowCmd{Playlist: "spotify:playlist:p1"}).Run(ctx); err != nil {
				t.Fatal(err)
			}
		}
		out.Reset()
		if err := (&PlaylistFollowingCmd{Playlist: "p1"}).Run(ctx); err != nil {
			t.Fatal(err)
		}
		if out.String() != want {
			t.Fatalf("output = %q, want %q", out.String(), want)
		}
	}
	if requests != 5 {
		t.Fatalf("requests = %d, want 5", requests)
	}
}

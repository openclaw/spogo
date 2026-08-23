package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestLibraryTracksAddCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	called := false
	mock := &testutil.SpotifyMock{
		LibraryModifyFn: func(ctx context.Context, path string, ids []string, method string) error {
			called = true
			if len(ids) != 1 || ids[0] != "t1" {
				t.Fatalf("unexpected ids: %#v", ids)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryTracksAddCmd{IDs: []string{"spotify:track:t1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatalf("expected call")
	}
}

func TestLibraryTracksAddCmdInvalid(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	cmd := LibraryTracksAddCmd{IDs: []string{"spotify:album:a1"}}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLibraryTracksListCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		LibraryTracksFn: func(ctx context.Context, limit, offset int) ([]spotify.Item, int, error) {
			return []spotify.Item{{ID: "t1", Name: "Track", Type: "track"}}, 1, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryTracksListCmd{Limit: 1}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestLibraryTracksListCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		LibraryTracksFn: func(ctx context.Context, limit, offset int) ([]spotify.Item, int, error) {
			return nil, 0, errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryTracksListCmd{Limit: 1}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLibraryTracksRemoveCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		LibraryModifyFn: func(ctx context.Context, path string, ids []string, method string) error {
			if method != "DELETE" {
				t.Fatalf("method %s", method)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryTracksRemoveCmd{IDs: []string{"spotify:track:t1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestLibraryArtistsListCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		FollowedArtistsFn: func(ctx context.Context, limit int, after string) ([]spotify.Item, int, string, error) {
			return []spotify.Item{{ID: "a1", Name: "Artist", Type: "artist"}}, 1, "", nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryArtistsListCmd{Limit: 1}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestLibraryArtistsListCmdOffsetError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	cmd := LibraryArtistsListCmd{Offset: 10}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLibraryAlbumsRemoveCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		LibraryModifyFn: func(ctx context.Context, path string, ids []string, method string) error {
			if path != "/me/albums" || method != "DELETE" {
				t.Fatalf("unexpected path/method")
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryAlbumsRemoveCmd{IDs: []string{"spotify:album:a1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestLibraryAlbumsListCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		LibraryAlbumsFn: func(ctx context.Context, limit, offset int) ([]spotify.Item, int, error) {
			return []spotify.Item{{ID: "a1", Name: "Album", Type: "album"}}, 1, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryAlbumsListCmd{Limit: 1}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestLibraryAlbumsListCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		LibraryAlbumsFn: func(ctx context.Context, limit, offset int) ([]spotify.Item, int, error) {
			return nil, 0, errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryAlbumsListCmd{Limit: 1}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLibraryAlbumsAddCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		LibraryModifyFn: func(ctx context.Context, path string, ids []string, method string) error {
			if path != "/me/albums" || method != "PUT" {
				t.Fatalf("unexpected path/method")
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryAlbumsAddCmd{IDs: []string{"spotify:album:a1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestLibraryArtistsFollowCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		FollowArtistsFn: func(ctx context.Context, ids []string, method string) error {
			if method != "PUT" || len(ids) != 1 || ids[0] != "a1" {
				t.Fatalf("unexpected follow")
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryArtistsFollowCmd{IDs: []string{"spotify:artist:a1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestLibraryArtistsUnfollowCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		FollowArtistsFn: func(ctx context.Context, ids []string, method string) error {
			if method != "DELETE" {
				t.Fatalf("method %s", method)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryArtistsUnfollowCmd{IDs: []string{"spotify:artist:a1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestLibraryPlaylistsListCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		PlaylistsFn: func(ctx context.Context, limit int, offset int) ([]spotify.Item, int, error) {
			return []spotify.Item{{ID: "p1", Name: "Playlist", Type: "playlist"}}, 1, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryPlaylistsListCmd{Limit: 1}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestLibraryPlaylistsListCmdHumanOutput(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatHuman)
	ctx.SetSpotify(&testutil.SpotifyMock{
		PlaylistsFn: func(context.Context, int, int) ([]spotify.Item, int, error) {
			return []spotify.Item{{ID: "p1", Name: "Discover Weekly", Type: "playlist", Owner: "Spotify"}}, 1, nil
		},
	})
	if err := (&LibraryPlaylistsListCmd{Limit: 1}).Run(ctx); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); !strings.Contains(got, "Discover Weekly — Spotify") {
		t.Fatalf("expected non-empty human playlist output, got %q", got)
	}
}

func TestLibraryPlaylistsListCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		PlaylistsFn: func(ctx context.Context, limit int, offset int) ([]spotify.Item, int, error) {
			return nil, 0, errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := LibraryPlaylistsListCmd{Limit: 1}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestPlayCmdArtistTopTrack(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		ArtistTopTracksFn: func(ctx context.Context, id string, limit int) ([]spotify.Item, error) {
			if id != "abc" {
				t.Fatalf("id %s", id)
			}
			if limit != 10 {
				t.Fatalf("limit %d", limit)
			}
			return []spotify.Item{{URI: "spotify:track:top"}}, nil
		},
		PlayFn: func(ctx context.Context, uri string) error {
			if uri != "spotify:track:top" {
				t.Fatalf("uri %s", uri)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlayCmd{Items: []string{"spotify:artist:abc"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestPlayCmdArtistFallbackSearch(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		ArtistTopTracksFn: func(context.Context, string, int) ([]spotify.Item, error) { return nil, errors.New("rate limit") },
		GetArtistFn:       func(context.Context, string) (spotify.Item, error) { return spotify.Item{Name: "Artist"}, nil },
		SearchFn: func(context.Context, string, string, int, int) (spotify.SearchResult, error) {
			return spotify.SearchResult{Items: []spotify.Item{{URI: "spotify:track:found"}}}, nil
		},
		PlayFn: func(context.Context, string) error { return nil },
	}
	ctx.SetSpotify(mock)
	if err := (&PlayCmd{Items: []string{"spotify:artist:abc"}}).Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestPlayCmdArtistFallbackArtistError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(&testutil.SpotifyMock{
		ArtistTopTracksFn: func(context.Context, string, int) ([]spotify.Item, error) { return nil, errors.New("boom") },
		GetArtistFn:       func(context.Context, string) (spotify.Item, error) { return spotify.Item{}, errors.New("missing") },
	})
	if err := (&PlayCmd{Items: []string{"spotify:artist:abc"}}).Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlayCmdArtistFallbackSearchEmpty(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(&testutil.SpotifyMock{
		ArtistTopTracksFn: func(context.Context, string, int) ([]spotify.Item, error) { return nil, nil },
		GetArtistFn:       func(context.Context, string) (spotify.Item, error) { return spotify.Item{Name: "Artist"}, nil },
		SearchFn: func(context.Context, string, string, int, int) (spotify.SearchResult, error) {
			return spotify.SearchResult{}, nil
		},
	})
	if err := (&PlayCmd{Items: []string{"spotify:artist:abc"}}).Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlayCmdArtistFallbackSearchError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(&testutil.SpotifyMock{
		ArtistTopTracksFn: func(context.Context, string, int) ([]spotify.Item, error) { return nil, errors.New("boom") },
		GetArtistFn:       func(context.Context, string) (spotify.Item, error) { return spotify.Item{Name: "Artist"}, nil },
		SearchFn: func(context.Context, string, string, int, int) (spotify.SearchResult, error) {
			return spotify.SearchResult{}, errors.New("search fail")
		},
	})
	if err := (&PlayCmd{Items: []string{"spotify:artist:abc"}}).Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

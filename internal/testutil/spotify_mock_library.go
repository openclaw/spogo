package testutil

import (
	"context"

	"github.com/steipete/spogo/internal/spotify"
)

func (m *SpotifyMock) LibraryTracks(ctx context.Context, limit, offset int) ([]spotify.Item, int, error) {
	if m.LibraryTracksFn == nil {
		return nil, 0, ErrNotImplemented
	}
	return m.LibraryTracksFn(ctx, limit, offset)
}

func (m *SpotifyMock) LibraryAlbums(ctx context.Context, limit, offset int) ([]spotify.Item, int, error) {
	if m.LibraryAlbumsFn == nil {
		return nil, 0, ErrNotImplemented
	}
	return m.LibraryAlbumsFn(ctx, limit, offset)
}

func (m *SpotifyMock) LibraryModify(ctx context.Context, path string, ids []string, method string) error {
	if m.LibraryModifyFn == nil {
		return ErrNotImplemented
	}
	return m.LibraryModifyFn(ctx, path, ids, method)
}

func (m *SpotifyMock) FollowArtists(ctx context.Context, ids []string, method string) error {
	if m.FollowArtistsFn == nil {
		return ErrNotImplemented
	}
	return m.FollowArtistsFn(ctx, ids, method)
}

func (m *SpotifyMock) FollowedArtists(ctx context.Context, limit int, after string) ([]spotify.Item, int, string, error) {
	if m.FollowedArtistsFn == nil {
		return nil, 0, "", ErrNotImplemented
	}
	return m.FollowedArtistsFn(ctx, limit, after)
}

func (m *SpotifyMock) Playlists(ctx context.Context, limit, offset int) ([]spotify.Item, int, error) {
	if m.PlaylistsFn == nil {
		return nil, 0, ErrNotImplemented
	}
	return m.PlaylistsFn(ctx, limit, offset)
}

func (m *SpotifyMock) PlaylistTracks(ctx context.Context, id string, limit, offset int) ([]spotify.Item, int, error) {
	if m.PlaylistTracksFn == nil {
		return nil, 0, ErrNotImplemented
	}
	return m.PlaylistTracksFn(ctx, id, limit, offset)
}

func (m *SpotifyMock) CreatePlaylist(ctx context.Context, name string, public, collaborative bool) (spotify.Item, error) {
	if m.CreatePlaylistFn == nil {
		return spotify.Item{}, ErrNotImplemented
	}
	return m.CreatePlaylistFn(ctx, name, public, collaborative)
}

func (m *SpotifyMock) AddTracks(ctx context.Context, playlistID string, uris []string) error {
	if m.AddTracksFn == nil {
		return ErrNotImplemented
	}
	return m.AddTracksFn(ctx, playlistID, uris)
}

func (m *SpotifyMock) RemoveTracks(ctx context.Context, playlistID string, uris []string) error {
	if m.RemoveTracksFn == nil {
		return ErrNotImplemented
	}
	return m.RemoveTracksFn(ctx, playlistID, uris)
}

func (m *SpotifyMock) GetUsersTopTracks(ctx context.Context, timeRange string, limit, offset int) (spotify.TopTracksResult, error) {
	if m.GetUsersTopTracksFn == nil {
		return spotify.TopTracksResult{}, ErrNotImplemented
	}
	return m.GetUsersTopTracksFn(ctx, timeRange, limit, offset)
}

func (m *SpotifyMock) GetRecentlyPlayed(ctx context.Context, limit int, after, before int64) (spotify.RecentlyPlayedResult, error) {
	if m.GetRecentlyPlayedFn == nil {
		return spotify.RecentlyPlayedResult{}, ErrNotImplemented
	}
	return m.GetRecentlyPlayedFn(ctx, limit, after, before)
}

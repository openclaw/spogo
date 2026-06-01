package testutil

import (
	"context"
	"errors"

	"github.com/steipete/spogo/internal/spotify"
)

var ErrNotImplemented = errors.New("not implemented")

type SpotifyMock struct {
	SearchFn            func(context.Context, string, string, int, int) (spotify.SearchResult, error)
	GetTrackFn          func(context.Context, string) (spotify.Item, error)
	GetAlbumFn          func(context.Context, string) (spotify.Item, error)
	GetArtistFn         func(context.Context, string) (spotify.Item, error)
	GetPlaylistFn       func(context.Context, string) (spotify.Item, error)
	GetShowFn           func(context.Context, string) (spotify.Item, error)
	GetEpisodeFn        func(context.Context, string) (spotify.Item, error)
	ArtistTopTracksFn   func(context.Context, string, int) ([]spotify.Item, error)
	PlaybackFn          func(context.Context) (spotify.PlaybackStatus, error)
	PlayFn              func(context.Context, string) error
	PauseFn             func(context.Context) error
	NextFn              func(context.Context) error
	PreviousFn          func(context.Context) error
	SeekFn              func(context.Context, int) error
	VolumeFn            func(context.Context, int) error
	ShuffleFn           func(context.Context, bool) error
	RepeatFn            func(context.Context, string) error
	DevicesFn           func(context.Context) ([]spotify.Device, error)
	TransferFn          func(context.Context, string) error
	QueueAddFn          func(context.Context, string) error
	QueueFn             func(context.Context) (spotify.Queue, error)
	LibraryTracksFn     func(context.Context, int, int) ([]spotify.Item, int, error)
	LibraryAlbumsFn     func(context.Context, int, int) ([]spotify.Item, int, error)
	LibraryModifyFn     func(context.Context, string, []string, string) error
	FollowArtistsFn     func(context.Context, []string, string) error
	FollowedArtistsFn   func(context.Context, int, string) ([]spotify.Item, int, string, error)
	PlaylistsFn         func(context.Context, int, int) ([]spotify.Item, int, error)
	PlaylistTracksFn    func(context.Context, string, int, int) ([]spotify.Item, int, error)
	CreatePlaylistFn    func(context.Context, string, bool, bool) (spotify.Item, error)
	AddTracksFn         func(context.Context, string, []string) error
	RemoveTracksFn      func(context.Context, string, []string) error
	GetUsersTopTracksFn func(context.Context, string, int, int) (spotify.TopTracksResult, error)
	GetRecentlyPlayedFn func(context.Context, int, int64, int64) (spotify.RecentlyPlayedResult, error)
}

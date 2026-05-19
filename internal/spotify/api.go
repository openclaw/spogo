package spotify

import "context"

type API interface {
	Search(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error)
	GetTrack(ctx context.Context, id string) (Item, error)
	GetAlbum(ctx context.Context, id string) (Item, error)
	GetArtist(ctx context.Context, id string) (Item, error)
	GetPlaylist(ctx context.Context, id string) (Item, error)
	GetShow(ctx context.Context, id string) (Item, error)
	GetEpisode(ctx context.Context, id string) (Item, error)
	Playback(ctx context.Context) (PlaybackStatus, error)
	Play(ctx context.Context, uri string) error
	Pause(ctx context.Context) error
	Next(ctx context.Context) error
	Previous(ctx context.Context) error
	Seek(ctx context.Context, positionMS int) error
	Volume(ctx context.Context, volume int) error
	Shuffle(ctx context.Context, enabled bool) error
	Repeat(ctx context.Context, mode string) error
	Devices(ctx context.Context) ([]Device, error)
	Transfer(ctx context.Context, deviceID string) error
	QueueAdd(ctx context.Context, uri string) error
	Queue(ctx context.Context) (Queue, error)
	LibraryTracks(ctx context.Context, limit, offset int) ([]Item, int, error)
	LibraryAlbums(ctx context.Context, limit, offset int) ([]Item, int, error)
	LibraryModify(ctx context.Context, path string, ids []string, method string) error
	FollowArtists(ctx context.Context, ids []string, method string) error
	FollowedArtists(ctx context.Context, limit int, after string) ([]Item, int, string, error)
	Playlists(ctx context.Context, limit, offset int) ([]Item, int, error)
	PlaylistTracks(ctx context.Context, id string, limit, offset int) ([]Item, int, error)
	CreatePlaylist(ctx context.Context, name string, public, collaborative bool) (Item, error)
	AddTracks(ctx context.Context, playlistID string, uris []string) error
	RemoveTracks(ctx context.Context, playlistID string, uris []string) error
	GetUsersTopTracks(ctx context.Context, timeRange string, limit, offset int) (TopTracksResult, error)
	GetRecentlyPlayed(ctx context.Context, limit int, after, before int64) (RecentlyPlayedResult, error)
}

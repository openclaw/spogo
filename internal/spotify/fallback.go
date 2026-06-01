package spotify

import (
	"context"
	"errors"
	"net/http"
)

type fallbackClient struct {
	web     API
	connect API
}

type artistTopTracksAPI interface {
	ArtistTopTracks(ctx context.Context, id string, limit int) ([]Item, error)
}

func NewPlaybackFallbackClient(web API, connect API) API {
	return &fallbackClient{web: web, connect: connect}
}

func (c *fallbackClient) shouldFallback(err error) bool {
	var apiErr APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusTooManyRequests
}

func fallbackCall[T any](c *fallbackClient, allow bool, fn func(API) (T, error)) (T, error) {
	res, err := fn(c.web)
	if err == nil || !allow || c.connect == nil || !c.shouldFallback(err) {
		return res, err
	}
	return fn(c.connect)
}

func fallbackVoid(c *fallbackClient, allow bool, fn func(API) error) error {
	err := fn(c.web)
	if err == nil || !allow || c.connect == nil || !c.shouldFallback(err) {
		return err
	}
	return fn(c.connect)
}

func (c *fallbackClient) Search(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	return fallbackCall(c, true, func(api API) (SearchResult, error) {
		return api.Search(ctx, kind, query, limit, offset)
	})
}

func (c *fallbackClient) ArtistTopTracks(ctx context.Context, id string, limit int) ([]Item, error) {
	web, ok := c.web.(artistTopTracksAPI)
	if !ok {
		return nil, ErrUnsupported
	}
	items, err := web.ArtistTopTracks(ctx, id, limit)
	if err == nil || c.connect == nil || !c.shouldFallback(err) {
		return items, err
	}
	if connect, ok := c.connect.(artistTopTracksAPI); ok {
		return connect.ArtistTopTracks(ctx, id, limit)
	}
	return items, err
}

func (c *fallbackClient) GetTrack(ctx context.Context, id string) (Item, error) {
	return fallbackCall(c, true, func(api API) (Item, error) {
		return api.GetTrack(ctx, id)
	})
}

func (c *fallbackClient) GetAlbum(ctx context.Context, id string) (Item, error) {
	return fallbackCall(c, true, func(api API) (Item, error) {
		return api.GetAlbum(ctx, id)
	})
}

func (c *fallbackClient) GetArtist(ctx context.Context, id string) (Item, error) {
	return fallbackCall(c, true, func(api API) (Item, error) {
		return api.GetArtist(ctx, id)
	})
}

func (c *fallbackClient) GetPlaylist(ctx context.Context, id string) (Item, error) {
	return fallbackCall(c, true, func(api API) (Item, error) {
		return api.GetPlaylist(ctx, id)
	})
}

func (c *fallbackClient) GetShow(ctx context.Context, id string) (Item, error) {
	return fallbackCall(c, true, func(api API) (Item, error) {
		return api.GetShow(ctx, id)
	})
}

func (c *fallbackClient) GetEpisode(ctx context.Context, id string) (Item, error) {
	return fallbackCall(c, true, func(api API) (Item, error) {
		return api.GetEpisode(ctx, id)
	})
}

func (c *fallbackClient) Playback(ctx context.Context) (PlaybackStatus, error) {
	return fallbackCall(c, true, func(api API) (PlaybackStatus, error) {
		return api.Playback(ctx)
	})
}

func (c *fallbackClient) Play(ctx context.Context, uri string) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.Play(ctx, uri)
	})
}

func (c *fallbackClient) Pause(ctx context.Context) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.Pause(ctx)
	})
}

func (c *fallbackClient) Next(ctx context.Context) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.Next(ctx)
	})
}

func (c *fallbackClient) Previous(ctx context.Context) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.Previous(ctx)
	})
}

func (c *fallbackClient) Seek(ctx context.Context, positionMS int) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.Seek(ctx, positionMS)
	})
}

func (c *fallbackClient) Volume(ctx context.Context, volume int) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.Volume(ctx, volume)
	})
}

func (c *fallbackClient) Shuffle(ctx context.Context, enabled bool) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.Shuffle(ctx, enabled)
	})
}

func (c *fallbackClient) Repeat(ctx context.Context, mode string) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.Repeat(ctx, mode)
	})
}

func (c *fallbackClient) Devices(ctx context.Context) ([]Device, error) {
	return fallbackCall(c, true, func(api API) ([]Device, error) {
		return api.Devices(ctx)
	})
}

func (c *fallbackClient) Transfer(ctx context.Context, deviceID string) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.Transfer(ctx, deviceID)
	})
}

func (c *fallbackClient) QueueAdd(ctx context.Context, uri string) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.QueueAdd(ctx, uri)
	})
}

func (c *fallbackClient) Queue(ctx context.Context) (Queue, error) {
	return fallbackCall(c, true, func(api API) (Queue, error) {
		return api.Queue(ctx)
	})
}

func (c *fallbackClient) LibraryTracks(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return c.web.LibraryTracks(ctx, limit, offset)
}

func (c *fallbackClient) LibraryAlbums(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return c.web.LibraryAlbums(ctx, limit, offset)
}

func (c *fallbackClient) LibraryModify(ctx context.Context, path string, ids []string, method string) error {
	return c.web.LibraryModify(ctx, path, ids, method)
}

func (c *fallbackClient) FollowArtists(ctx context.Context, ids []string, method string) error {
	return c.web.FollowArtists(ctx, ids, method)
}

func (c *fallbackClient) FollowedArtists(ctx context.Context, limit int, after string) ([]Item, int, string, error) {
	return c.web.FollowedArtists(ctx, limit, after)
}

func (c *fallbackClient) Playlists(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return c.web.Playlists(ctx, limit, offset)
}

func (c *fallbackClient) PlaylistTracks(ctx context.Context, id string, limit, offset int) ([]Item, int, error) {
	return c.web.PlaylistTracks(ctx, id, limit, offset)
}

func (c *fallbackClient) CreatePlaylist(ctx context.Context, name string, public, collaborative bool) (Item, error) {
	return c.web.CreatePlaylist(ctx, name, public, collaborative)
}

func (c *fallbackClient) AddTracks(ctx context.Context, playlistID string, uris []string) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.AddTracks(ctx, playlistID, uris)
	})
}

func (c *fallbackClient) RemoveTracks(ctx context.Context, playlistID string, uris []string) error {
	return fallbackVoid(c, true, func(api API) error {
		return api.RemoveTracks(ctx, playlistID, uris)
	})
}

func (c *fallbackClient) GetUsersTopTracks(ctx context.Context, timeRange string, limit, offset int) (TopTracksResult, error) {
	return fallbackCall(c, true, func(api API) (TopTracksResult, error) {
		return api.GetUsersTopTracks(ctx, timeRange, limit, offset)
	})
}

func (c *fallbackClient) GetRecentlyPlayed(ctx context.Context, limit int, after, before int64) (RecentlyPlayedResult, error) {
	return fallbackCall(c, true, func(api API) (RecentlyPlayedResult, error) {
		return api.GetRecentlyPlayed(ctx, limit, after, before)
	})
}

package spotify

import (
	"context"
	"errors"
	"net/http"
)

type autoClient struct {
	primary   API
	secondary API
}

func NewAutoClient(connect API, web API) API {
	return &autoClient{primary: connect, secondary: web}
}

func (c *autoClient) shouldFallback(err error) bool {
	if errors.Is(err, ErrUnsupported) {
		return true
	}
	var apiErr APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusTooManyRequests
}

func autoCall[T any](c *autoClient, fn func(API) (T, error)) (T, error) {
	res, err := fn(c.primary)
	if err == nil || c.secondary == nil || !c.shouldFallback(err) {
		return res, err
	}
	return fn(c.secondary)
}

func autoVoid(c *autoClient, fn func(API) error) error {
	err := fn(c.primary)
	if err == nil || c.secondary == nil || !c.shouldFallback(err) {
		return err
	}
	return fn(c.secondary)
}

func autoCall2[A, B any](c *autoClient, fn func(API) (A, B, error)) (A, B, error) {
	a, b, err := fn(c.primary)
	if err == nil || c.secondary == nil || !c.shouldFallback(err) {
		return a, b, err
	}
	return fn(c.secondary)
}

func autoCall3[A, B, C any](c *autoClient, fn func(API) (A, B, C, error)) (A, B, C, error) {
	a, b, cval, err := fn(c.primary)
	if err == nil || c.secondary == nil || !c.shouldFallback(err) {
		return a, b, cval, err
	}
	return fn(c.secondary)
}

func (c *autoClient) Search(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	return autoCall(c, func(api API) (SearchResult, error) {
		return api.Search(ctx, kind, query, limit, offset)
	})
}

func (c *autoClient) ArtistTopTracks(ctx context.Context, id string, limit int) ([]Item, error) {
	if primary, ok := c.primary.(artistTopTracksAPI); ok {
		items, err := primary.ArtistTopTracks(ctx, id, limit)
		if err == nil || c.secondary == nil || !c.shouldFallback(err) {
			return items, err
		}
	}
	if secondary, ok := c.secondary.(artistTopTracksAPI); ok {
		return secondary.ArtistTopTracks(ctx, id, limit)
	}
	return nil, ErrUnsupported
}

func (c *autoClient) GetTrack(ctx context.Context, id string) (Item, error) {
	return autoCall(c, func(api API) (Item, error) {
		return api.GetTrack(ctx, id)
	})
}

func (c *autoClient) GetAlbum(ctx context.Context, id string) (Item, error) {
	return autoCall(c, func(api API) (Item, error) {
		return api.GetAlbum(ctx, id)
	})
}

func (c *autoClient) GetArtist(ctx context.Context, id string) (Item, error) {
	return autoCall(c, func(api API) (Item, error) {
		return api.GetArtist(ctx, id)
	})
}

func (c *autoClient) GetPlaylist(ctx context.Context, id string) (Item, error) {
	return autoCall(c, func(api API) (Item, error) {
		return api.GetPlaylist(ctx, id)
	})
}

func (c *autoClient) GetShow(ctx context.Context, id string) (Item, error) {
	return autoCall(c, func(api API) (Item, error) {
		return api.GetShow(ctx, id)
	})
}

func (c *autoClient) GetEpisode(ctx context.Context, id string) (Item, error) {
	return autoCall(c, func(api API) (Item, error) {
		return api.GetEpisode(ctx, id)
	})
}

func (c *autoClient) Playback(ctx context.Context) (PlaybackStatus, error) {
	return autoCall(c, func(api API) (PlaybackStatus, error) {
		return api.Playback(ctx)
	})
}

func (c *autoClient) Play(ctx context.Context, uri string) error {
	return autoVoid(c, func(api API) error {
		return api.Play(ctx, uri)
	})
}

func (c *autoClient) Pause(ctx context.Context) error {
	return autoVoid(c, func(api API) error {
		return api.Pause(ctx)
	})
}

func (c *autoClient) Next(ctx context.Context) error {
	return autoVoid(c, func(api API) error {
		return api.Next(ctx)
	})
}

func (c *autoClient) Previous(ctx context.Context) error {
	return autoVoid(c, func(api API) error {
		return api.Previous(ctx)
	})
}

func (c *autoClient) Seek(ctx context.Context, positionMS int) error {
	return autoVoid(c, func(api API) error {
		return api.Seek(ctx, positionMS)
	})
}

func (c *autoClient) Volume(ctx context.Context, volume int) error {
	return autoVoid(c, func(api API) error {
		return api.Volume(ctx, volume)
	})
}

func (c *autoClient) Shuffle(ctx context.Context, enabled bool) error {
	return autoVoid(c, func(api API) error {
		return api.Shuffle(ctx, enabled)
	})
}

func (c *autoClient) Repeat(ctx context.Context, mode string) error {
	return autoVoid(c, func(api API) error {
		return api.Repeat(ctx, mode)
	})
}

func (c *autoClient) Devices(ctx context.Context) ([]Device, error) {
	return autoCall(c, func(api API) ([]Device, error) {
		return api.Devices(ctx)
	})
}

func (c *autoClient) Transfer(ctx context.Context, deviceID string) error {
	return autoVoid(c, func(api API) error {
		return api.Transfer(ctx, deviceID)
	})
}

func (c *autoClient) QueueAdd(ctx context.Context, uri string) error {
	return autoVoid(c, func(api API) error {
		return api.QueueAdd(ctx, uri)
	})
}

func (c *autoClient) Queue(ctx context.Context) (Queue, error) {
	return autoCall(c, func(api API) (Queue, error) {
		return api.Queue(ctx)
	})
}

func (c *autoClient) LibraryTracks(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return autoCall2(c, func(api API) ([]Item, int, error) {
		return api.LibraryTracks(ctx, limit, offset)
	})
}

func (c *autoClient) LibraryAlbums(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return autoCall2(c, func(api API) ([]Item, int, error) {
		return api.LibraryAlbums(ctx, limit, offset)
	})
}

func (c *autoClient) LibraryModify(ctx context.Context, path string, ids []string, method string) error {
	return autoVoid(c, func(api API) error {
		return api.LibraryModify(ctx, path, ids, method)
	})
}

func (c *autoClient) FollowArtists(ctx context.Context, ids []string, method string) error {
	return autoVoid(c, func(api API) error {
		return api.FollowArtists(ctx, ids, method)
	})
}

func (c *autoClient) FollowedArtists(ctx context.Context, limit int, after string) ([]Item, int, string, error) {
	return autoCall3(c, func(api API) ([]Item, int, string, error) {
		return api.FollowedArtists(ctx, limit, after)
	})
}

func (c *autoClient) Playlists(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return autoCall2(c, func(api API) ([]Item, int, error) {
		return api.Playlists(ctx, limit, offset)
	})
}

func (c *autoClient) PlaylistTracks(ctx context.Context, id string, limit, offset int) ([]Item, int, error) {
	return autoCall2(c, func(api API) ([]Item, int, error) {
		return api.PlaylistTracks(ctx, id, limit, offset)
	})
}

func (c *autoClient) CreatePlaylist(ctx context.Context, name string, public, collaborative bool) (Item, error) {
	return autoCall(c, func(api API) (Item, error) {
		return api.CreatePlaylist(ctx, name, public, collaborative)
	})
}

func (c *autoClient) AddTracks(ctx context.Context, playlistID string, uris []string) error {
	return autoVoid(c, func(api API) error {
		return api.AddTracks(ctx, playlistID, uris)
	})
}

func (c *autoClient) RemoveTracks(ctx context.Context, playlistID string, uris []string) error {
	return autoVoid(c, func(api API) error {
		return api.RemoveTracks(ctx, playlistID, uris)
	})
}

func (c *autoClient) GetUsersTopTracks(ctx context.Context, timeRange string, limit, offset int) (TopTracksResult, error) {
	return autoCall(c, func(api API) (TopTracksResult, error) {
		return api.GetUsersTopTracks(ctx, timeRange, limit, offset)
	})
}

func (c *autoClient) GetRecentlyPlayed(ctx context.Context, limit int, after, before int64) (RecentlyPlayedResult, error) {
	return autoCall(c, func(api API) (RecentlyPlayedResult, error) {
		return api.GetRecentlyPlayed(ctx, limit, after, before)
	})
}

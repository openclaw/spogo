package spotify

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/steipete/spogo/internal/cookies"
)

type ConnectOptions struct {
	Source    cookies.Source
	Market    string
	Language  string
	Device    string
	Timeout   time.Duration
	CachePath string
	WebClient *Client
}

type ConnectClient struct {
	source       cookies.Source
	market       string
	language     string
	device       string
	client       *http.Client
	session      *connectSession
	hashes       *hashResolver
	webMu        sync.Mutex
	web          *Client
	searchClient *http.Client

	cache *connectCacheStore

	routeMu              sync.RWMutex
	cachedActiveDeviceID string
	cachedOriginDeviceID string
	cachedRouteAt        time.Time
}

func NewConnectClient(opts ConnectOptions) (*ConnectClient, error) {
	if opts.Source == nil {
		return nil, errors.New("cookie source required")
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = defaultHTTPClientTimeout
	}
	httpClient := &http.Client{Timeout: timeout}
	cache := newConnectCacheStore(opts.CachePath)
	session := &connectSession{
		source: opts.Source,
		client: httpClient,
		cache:  cache,
	}
	return &ConnectClient{
		source:   opts.Source,
		market:   opts.Market,
		language: opts.Language,
		device:   opts.Device,
		client:   httpClient,
		session:  session,
		hashes:   newHashResolver(httpClient, session),
		cache:    cache,
		web:      opts.WebClient,
	}, nil
}

func (c *ConnectClient) Search(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	return c.search(ctx, kind, query, limit, offset)
}

func (c *ConnectClient) GetTrack(ctx context.Context, id string) (Item, error) {
	return c.trackInfo(ctx, id)
}

func (c *ConnectClient) GetAlbum(ctx context.Context, id string) (Item, error) {
	return c.albumInfo(ctx, id)
}

func (c *ConnectClient) GetArtist(ctx context.Context, id string) (Item, error) {
	return c.artistInfo(ctx, id)
}

func (c *ConnectClient) GetPlaylist(ctx context.Context, id string) (Item, error) {
	return c.playlistInfo(ctx, id)
}

func (c *ConnectClient) GetShow(ctx context.Context, id string) (Item, error) {
	return c.showInfo(ctx, id)
}

func (c *ConnectClient) GetEpisode(ctx context.Context, id string) (Item, error) {
	return c.episodeInfo(ctx, id)
}

func (c *ConnectClient) Playback(ctx context.Context) (PlaybackStatus, error) {
	return c.playback(ctx)
}

func (c *ConnectClient) Play(ctx context.Context, uri string) error {
	return c.play(ctx, uri)
}

func (c *ConnectClient) Pause(ctx context.Context) error {
	return c.pause(ctx)
}

func (c *ConnectClient) Next(ctx context.Context) error {
	return c.next(ctx)
}

func (c *ConnectClient) Previous(ctx context.Context) error {
	return c.previous(ctx)
}

func (c *ConnectClient) Seek(ctx context.Context, positionMS int) error {
	return c.seek(ctx, positionMS)
}

func (c *ConnectClient) Volume(ctx context.Context, volume int) error {
	return c.volume(ctx, volume)
}

func (c *ConnectClient) Shuffle(ctx context.Context, enabled bool) error {
	return c.shuffle(ctx, enabled)
}

func (c *ConnectClient) Repeat(ctx context.Context, mode string) error {
	return c.repeat(ctx, mode)
}

func (c *ConnectClient) Devices(ctx context.Context) ([]Device, error) {
	return c.devices(ctx)
}

func (c *ConnectClient) Transfer(ctx context.Context, deviceID string) error {
	return c.transfer(ctx, deviceID)
}

func (c *ConnectClient) QueueAdd(ctx context.Context, uri string) error {
	return c.queueAdd(ctx, uri)
}

func (c *ConnectClient) Queue(ctx context.Context) (Queue, error) {
	return c.queue(ctx)
}

func (c *ConnectClient) LibraryTracks(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return withWebCollectionFallback(c, func() ([]Item, int, error) {
		return c.libraryTracks(ctx, limit, offset)
	}, func(web *Client) ([]Item, int, error) {
		return web.LibraryTracks(ctx, limit, offset)
	})
}

func (c *ConnectClient) LibraryAlbums(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return withWebCollectionFallback(c, func() ([]Item, int, error) {
		return c.libraryAlbums(ctx, limit, offset)
	}, func(web *Client) ([]Item, int, error) {
		return web.LibraryAlbums(ctx, limit, offset)
	})
}

func (c *ConnectClient) LibraryModify(ctx context.Context, path string, ids []string, method string) error {
	return withWebFallback(c, func(web *Client) error {
		return web.LibraryModify(ctx, path, ids, method)
	})
}

func (c *ConnectClient) FollowArtists(ctx context.Context, ids []string, method string) error {
	return withWebFallback(c, func(web *Client) error {
		return web.FollowArtists(ctx, ids, method)
	})
}

func (c *ConnectClient) FollowedArtists(ctx context.Context, limit int, after string) ([]Item, int, string, error) {
	items, total, next, err := c.followedArtists(ctx, limit, after)
	if err == nil {
		return items, total, next, nil
	}
	return withWebCursorFallback(c, func(web *Client) ([]Item, int, string, error) {
		return web.FollowedArtists(ctx, limit, after)
	})
}

func (c *ConnectClient) Playlists(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return withWebCollectionFallback(c, func() ([]Item, int, error) {
		return c.playlists(ctx, limit, offset)
	}, func(web *Client) ([]Item, int, error) {
		return web.Playlists(ctx, limit, offset)
	})
}

func (c *ConnectClient) PlaylistTracks(ctx context.Context, id string, limit, offset int) ([]Item, int, error) {
	return withWebCollectionFallback(c, func() ([]Item, int, error) {
		return c.playlistTracks(ctx, id, limit, offset)
	}, func(web *Client) ([]Item, int, error) {
		return web.PlaylistTracks(ctx, id, limit, offset)
	})
}

func (c *ConnectClient) CreatePlaylist(ctx context.Context, name string, public, collaborative bool) (Item, error) {
	return withWebItemFallback(c, func(web *Client) (Item, error) {
		return web.CreatePlaylist(ctx, name, public, collaborative)
	})
}

func (c *ConnectClient) AddTracks(ctx context.Context, playlistID string, uris []string) error {
	if err := c.addTracks(ctx, playlistID, uris); err == nil {
		return nil
	} else if errors.Is(err, errPlaylistNotWritable) {
		return err
	}
	return withWebFallback(c, func(web *Client) error {
		return web.AddTracks(ctx, playlistID, uris)
	})
}

func (c *ConnectClient) RemoveTracks(ctx context.Context, playlistID string, uris []string) error {
	if err := c.removeTracks(ctx, playlistID, uris); err == nil {
		return nil
	} else if errors.Is(err, errPlaylistNotWritable) {
		return err
	}
	return withWebFallback(c, func(web *Client) error {
		return web.RemoveTracks(ctx, playlistID, uris)
	})
}

func (c *ConnectClient) GetUsersTopTracks(ctx context.Context, timeRange string, limit, offset int) (TopTracksResult, error) {
	return c.userTopTracks(ctx, timeRange, limit, offset)
}

func (c *ConnectClient) GetRecentlyPlayed(ctx context.Context, limit int, after, before int64) (RecentlyPlayedResult, error) {
	return c.recentlyPlayed(ctx, limit, after, before)
}

func withWebCollectionFallback(c *ConnectClient, primary func() ([]Item, int, error), fallback func(*Client) ([]Item, int, error)) ([]Item, int, error) {
	items, total, err := primary()
	if err == nil {
		return items, total, nil
	}
	web, werr := c.webClient()
	if werr != nil {
		return nil, 0, err
	}
	fallbackItems, fallbackTotal, fallbackErr := fallback(web)
	return fallbackItems, fallbackTotal, preserveRateLimitHint(err, fallbackErr)
}

func withWebCursorFallback(c *ConnectClient, fallback func(*Client) ([]Item, int, string, error)) ([]Item, int, string, error) {
	web, err := c.webClient()
	if err != nil {
		return nil, 0, "", err
	}
	return fallback(web)
}

func withWebItemFallback(c *ConnectClient, fallback func(*Client) (Item, error)) (Item, error) {
	web, err := c.webClient()
	if err != nil {
		return Item{}, err
	}
	return fallback(web)
}

func withWebFallback(c *ConnectClient, fallback func(*Client) error) error {
	web, err := c.webClient()
	if err != nil {
		return err
	}
	return fallback(web)
}

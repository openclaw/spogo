package spotify

import (
	"context"
	"fmt"
)

func (c *ConnectClient) trackInfo(ctx context.Context, id string) (Item, error) {
	return c.infoWithWebFallback(ctx, id, "track", func() (Item, error) {
		return c.infoByOperation(ctx, "getTrack", map[string]any{"uri": "spotify:track:" + id}, "track")
	}, func(web *Client) (Item, error) {
		return web.GetTrack(ctx, id)
	})
}

func (c *ConnectClient) albumInfo(ctx context.Context, id string) (Item, error) {
	return c.infoWithWebFallback(ctx, id, "album", func() (Item, error) {
		return c.infoByOperation(ctx, "getAlbum", map[string]any{
			"uri":    "spotify:album:" + id,
			"locale": c.language,
			"offset": 0,
			"limit":  50,
		}, "album")
	}, func(web *Client) (Item, error) {
		return web.GetAlbum(ctx, id)
	})
}

func (c *ConnectClient) artistInfo(ctx context.Context, id string) (Item, error) {
	return c.infoWithWebFallback(ctx, id, "artist", func() (Item, error) {
		return c.infoByOperation(ctx, "queryArtistOverview", map[string]any{
			"uri":    "spotify:artist:" + id,
			"locale": c.language,
		}, "artist")
	}, func(web *Client) (Item, error) {
		return web.GetArtist(ctx, id)
	})
}

func (c *ConnectClient) playlistInfo(ctx context.Context, id string) (Item, error) {
	return c.infoWithWebFallback(ctx, id, "playlist", func() (Item, error) {
		return c.infoByOperation(ctx, "fetchPlaylist", map[string]any{
			"uri":                       "spotify:playlist:" + id,
			"offset":                    0,
			"limit":                     25,
			"enableWatchFeedEntrypoint": false,
		}, "playlist")
	}, func(web *Client) (Item, error) {
		return web.GetPlaylist(ctx, id)
	})
}

func (c *ConnectClient) showInfo(ctx context.Context, id string) (Item, error) {
	return c.infoWithWebFallback(ctx, id, "show", func() (Item, error) {
		item, err := c.infoByOperation(ctx, "queryPodcastEpisodes", map[string]any{
			"uri":                            "spotify:show:" + id,
			"offset":                         0,
			"limit":                          25,
			"includeEpisodeContentRatingsV2": false,
		}, "show")
		if err != nil || item.Publisher != "" || item.Name == "" {
			return item, err
		}
		if payload, searchErr := c.graphQL(ctx, "searchDesktop", searchVariables(item.Name, 10, 0)); searchErr == nil {
			matches, _ := extractSearchItems(payload, "show")
			for _, match := range matches {
				if match.URI == item.URI {
					item.Publisher = match.Publisher
					break
				}
			}
		}
		return item, nil
	}, func(web *Client) (Item, error) {
		return web.GetShow(ctx, id)
	})
}

func (c *ConnectClient) episodeInfo(ctx context.Context, id string) (Item, error) {
	return c.infoWithWebFallback(ctx, id, "episode", func() (Item, error) {
		return c.infoByOperation(ctx, "getEpisodeOrChapter", map[string]any{
			"uri":                            "spotify:episode:" + id,
			"includeEpisodeContentRatingsV2": false,
		}, "episode")
	}, func(web *Client) (Item, error) {
		return web.GetEpisode(ctx, id)
	})
}

func (c *ConnectClient) ArtistTopTracks(ctx context.Context, id string, limit int) ([]Item, error) {
	web, err := c.webClient()
	if err != nil {
		return nil, err
	}
	return web.ArtistTopTracks(ctx, id, limit)
}

func (c *ConnectClient) infoByOperation(ctx context.Context, operation string, variables map[string]any, kind string) (Item, error) {
	payload, err := c.graphQL(ctx, operation, variables)
	if err != nil {
		return Item{}, err
	}
	item, ok := extractRequestedItemFromPayload(payload, kind, getString(variables, "uri"))
	if !ok {
		return Item{}, fmt.Errorf("no %s found", kind)
	}
	return item, nil
}

func (c *ConnectClient) infoWithWebFallback(ctx context.Context, id, kind string, connectLookup func() (Item, error), webLookup func(*Client) (Item, error)) (Item, error) {
	item, err := connectLookup()
	if err == nil {
		return item, nil
	}
	web, werr := c.webClient()
	if werr != nil {
		return Item{}, err
	}
	fallback, fallbackErr := webLookup(web)
	return fallback, preserveRateLimitHint(err, fallbackErr)
}

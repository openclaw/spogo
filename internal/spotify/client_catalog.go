package spotify

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) Search(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("type", kind)
	params.Set("limit", fmt.Sprint(limit))
	params.Set("offset", fmt.Sprint(offset))
	var response map[string]searchContainer
	if err := c.get(ctx, "/search", params, &response); err != nil {
		return SearchResult{}, err
	}
	container, ok := response[kind+"s"]
	if !ok {
		return SearchResult{}, fmt.Errorf("missing %s result", kind)
	}
	items := make([]Item, 0, len(container.Items))
	for _, raw := range container.Items {
		item, err := mapSearchItem(kind, raw)
		if err != nil {
			return SearchResult{}, err
		}
		items = append(items, item)
	}
	return SearchResult{
		Type:   kind,
		Limit:  container.Limit,
		Offset: container.Offset,
		Total:  container.Total,
		Items:  items,
	}, nil
}

func (c *Client) GetTrack(ctx context.Context, id string) (Item, error) {
	var raw trackItem
	if err := c.get(ctx, "/tracks/"+id, url.Values{}, &raw); err != nil {
		return Item{}, err
	}
	return mapTrack(raw), nil
}

func (c *Client) GetAlbum(ctx context.Context, id string) (Item, error) {
	var raw albumItem
	if err := c.get(ctx, "/albums/"+id, url.Values{}, &raw); err != nil {
		return Item{}, err
	}
	return mapAlbum(raw), nil
}

func (c *Client) GetArtist(ctx context.Context, id string) (Item, error) {
	var raw artistItem
	if err := c.get(ctx, "/artists/"+id, nil, &raw); err != nil {
		return Item{}, err
	}
	return mapArtist(raw), nil
}

func (c *Client) GetPlaylist(ctx context.Context, id string) (Item, error) {
	var raw playlistItem
	if err := c.get(ctx, "/playlists/"+id, nil, &raw); err != nil {
		return Item{}, err
	}
	return mapPlaylist(raw), nil
}

func (c *Client) GetShow(ctx context.Context, id string) (Item, error) {
	var raw showItem
	if err := c.get(ctx, "/shows/"+id, url.Values{}, &raw); err != nil {
		return Item{}, err
	}
	return mapShow(raw), nil
}

func (c *Client) GetEpisode(ctx context.Context, id string) (Item, error) {
	var raw episodeItem
	if err := c.get(ctx, "/episodes/"+id, url.Values{}, &raw); err != nil {
		return Item{}, err
	}
	return mapEpisode(raw), nil
}

func (c *Client) ArtistTopTracks(ctx context.Context, id string, limit int) ([]Item, error) {
	params := url.Values{}
	market := c.market
	if market == "" {
		market = "US"
	}
	params.Set("market", market)
	var raw artistTopTracksResponse
	if err := c.get(ctx, "/artists/"+id+"/top-tracks", params, &raw); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > len(raw.Tracks) {
		limit = len(raw.Tracks)
	}
	items := make([]Item, 0, limit)
	for i := 0; i < limit; i++ {
		items = append(items, mapTrack(raw.Tracks[i]))
	}
	return items, nil
}

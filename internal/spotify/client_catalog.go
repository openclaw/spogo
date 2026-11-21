package spotify

import (
	"context"
	"fmt"
	"net/url"
)

// searchPageLimit is the largest page one /search request may ask for in
// Spotify's Development Mode. Larger requests are served by paging, so the
// caller's result count survives the reduced ceiling.
const searchPageLimit = 10

// searchMaxOffset is the highest offset Spotify accepts on /search. Paging
// stops there instead of requesting a page the API will reject.
const searchMaxOffset = 1000

func (c *Client) Search(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	if limit <= 0 {
		limit = searchPageLimit
	}
	result := SearchResult{Type: kind, Limit: limit, Offset: offset, Items: make([]Item, 0, limit)}
	remaining := limit
	pageOffset := offset
	first := true
	for remaining > 0 && pageOffset <= searchMaxOffset {
		pageSize := remaining
		if pageSize > searchPageLimit {
			pageSize = searchPageLimit
		}
		container, err := c.searchPage(ctx, kind, query, pageSize, pageOffset)
		if err != nil {
			return SearchResult{}, err
		}
		if first {
			result.Total = container.Total
			first = false
		}
		for _, raw := range container.Items {
			item, err := mapSearchItem(kind, raw)
			if err != nil {
				return SearchResult{}, err
			}
			result.Items = append(result.Items, item)
		}
		if len(container.Items) < pageSize {
			break
		}
		remaining -= len(container.Items)
		pageOffset += len(container.Items)
	}
	return result, nil
}

func (c *Client) searchPage(ctx context.Context, kind, query string, limit, offset int) (searchContainer, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("type", kind)
	params.Set("limit", fmt.Sprint(limit))
	params.Set("offset", fmt.Sprint(offset))
	var response map[string]searchContainer
	if err := c.get(ctx, "/search", params, &response); err != nil {
		return searchContainer{}, err
	}
	container, ok := response[kind+"s"]
	if !ok {
		return searchContainer{}, fmt.Errorf("missing %s result", kind)
	}
	return container, nil
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

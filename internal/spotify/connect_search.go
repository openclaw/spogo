package spotify

import (
	"context"
	"errors"
	"strings"
	"sync"
)

const searchMetadataConcurrency = 6

func (c *ConnectClient) search(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return SearchResult{}, errors.New("query required")
	}
	limit = normalizeSearchLimit(limit)
	offset = normalizeOffset(offset)
	payload, err := c.graphQL(ctx, "searchDesktop", searchVariables(query, limit, offset))
	if err != nil {
		fallback, ferr := c.searchViaWeb(ctx, kind, query, limit, offset)
		if ferr == nil {
			return fallback, nil
		}
		return SearchResult{}, preserveRateLimitHint(err, ferr)
	}
	items, total := extractSearchItems(payload, kind)
	c.hydrateSearchItems(ctx, kind, items)
	return SearchResult{Type: kind, Limit: limit, Offset: offset, Total: total, Items: items}, nil
}

func (c *ConnectClient) hydrateSearchItems(ctx context.Context, kind string, items []Item) {
	if kind != "album" && kind != "artist" && kind != "playlist" && kind != "show" {
		return
	}
	jobs := make(chan int)
	var workers sync.WaitGroup
	for range min(searchMetadataConcurrency, len(items)) {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				if detailed, err := c.searchItemDetails(ctx, kind, items[index].ID); err == nil {
					items[index] = mergeSearchItem(items[index], detailed)
				}
			}
		}()
	}
	for index := range items {
		select {
		case jobs <- index:
		case <-ctx.Done():
			close(jobs)
			workers.Wait()
			return
		}
	}
	close(jobs)
	workers.Wait()
}

func (c *ConnectClient) searchItemDetails(ctx context.Context, kind, id string) (Item, error) {
	uri := "spotify:" + kind + ":" + id
	switch kind {
	case "album":
		return c.infoByOperation(ctx, "getAlbum", map[string]any{"uri": uri, "locale": c.language, "offset": 0, "limit": 1}, kind)
	case "artist":
		return c.infoByOperation(ctx, "queryArtistOverview", map[string]any{"uri": uri, "locale": c.language}, kind)
	case "playlist":
		return c.infoByOperation(ctx, "fetchPlaylist", map[string]any{"uri": uri, "offset": 0, "limit": 1, "enableWatchFeedEntrypoint": false}, kind)
	case "show":
		return c.infoByOperation(ctx, "queryPodcastEpisodes", map[string]any{"uri": uri, "offset": 0, "limit": 1, "includeEpisodeContentRatingsV2": false}, kind)
	default:
		return Item{}, ErrUnsupported
	}
}

func mergeSearchItem(item, detailed Item) Item {
	if detailed.Name != "" {
		item.Name = detailed.Name
	}
	if len(detailed.Artists) > 0 {
		item.Artists = detailed.Artists
	}
	if detailed.Owner != "" {
		item.Owner = detailed.Owner
	}
	if detailed.ReleaseDate != "" {
		item.ReleaseDate = detailed.ReleaseDate
	}
	if detailed.TotalTracks > 0 {
		item.TotalTracks = detailed.TotalTracks
	}
	if detailed.TotalEpisodes > 0 {
		item.TotalEpisodes = detailed.TotalEpisodes
	}
	if detailed.Followers > 0 {
		item.Followers = detailed.Followers
	}
	if detailed.Publisher != "" {
		item.Publisher = detailed.Publisher
	}
	if len(detailed.Genres) > 0 {
		item.Genres = detailed.Genres
	}
	return item
}

func (c *ConnectClient) searchViaWeb(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	return c.searchViaWebAPI(ctx, kind, query, limit, offset)
}

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func searchVariables(query string, limit, offset int) map[string]any {
	return map[string]any{
		"searchTerm":                    query,
		"offset":                        offset,
		"limit":                         limit,
		"numberOfTopResults":            5,
		"includeAudiobooks":             true,
		"includePreReleases":            true,
		"includeLocalConcertsField":     false,
		"includeArtistHasConcertsField": false,
	}
}

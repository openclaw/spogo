package spotify

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) GetUsersTopTracks(ctx context.Context, timeRange string, limit, offset int) (TopTracksResult, error) {
	params := url.Values{}
	params.Set("time_range", timeRange)
	params.Set("limit", fmt.Sprint(limit))
	params.Set("offset", fmt.Sprint(offset))
	var raw topTracksResponse
	if err := c.get(ctx, "/me/top/tracks", params, &raw); err != nil {
		return TopTracksResult{}, err
	}
	items := make([]Item, 0, len(raw.Items))
	for _, t := range raw.Items {
		items = append(items, mapTrack(t))
	}
	return TopTracksResult{
		Total:  raw.Total,
		Limit:  raw.Limit,
		Offset: raw.Offset,
		Items:  items,
	}, nil
}

func (c *Client) GetRecentlyPlayed(ctx context.Context, limit int, after, before int64) (RecentlyPlayedResult, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprint(limit))
	if after > 0 {
		params.Set("after", fmt.Sprint(after))
	} else if before > 0 {
		params.Set("before", fmt.Sprint(before))
	}
	var raw recentlyPlayedResponse
	if err := c.get(ctx, "/me/player/recently-played", params, &raw); err != nil {
		return RecentlyPlayedResult{}, err
	}
	items := make([]RecentlyPlayedItem, 0, len(raw.Items))
	for _, item := range raw.Items {
		items = append(items, RecentlyPlayedItem{
			Track:    mapTrack(item.Track),
			PlayedAt: item.PlayedAt,
		})
	}
	result := RecentlyPlayedResult{
		Items: items,
		Next:  raw.Next,
		Limit: raw.Limit,
	}
	if raw.Cursors != nil {
		result.Cursors = &Cursors{
			After:  raw.Cursors.After,
			Before: raw.Cursors.Before,
		}
	}
	return result, nil
}

package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	recentlyPlayedBaseURL = "https://spclient.wg.spotify.com/recently-played/v3"
	recentlyPlayedPage    = 50
)

type recentlyPlayedContextsResponse struct {
	PlayContexts []recentlyPlayedContext `json:"playContexts"`
}

type recentlyPlayedContext struct {
	URI                string `json:"uri"`
	LastPlayedTime     int64  `json:"lastPlayedTime"`
	LastPlayedTrackURI string `json:"lastPlayedTrackUri"`
}

func (c *ConnectClient) userTopTracks(ctx context.Context, timeRange string, limit, offset int) (TopTracksResult, error) {
	vars := map[string]any{
		"includeTopArtists": false,
		"topArtistsInput": map[string]any{
			"offset":    0,
			"limit":     0,
			"sortBy":    "AFFINITY",
			"timeRange": connectTopTimeRange(timeRange),
		},
		"includeTopTracks": true,
		"topTracksInput": map[string]any{
			"offset":    offset,
			"limit":     limit,
			"sortBy":    "AFFINITY",
			"timeRange": connectTopTimeRange(timeRange),
		},
	}
	payload, err := c.graphQL(ctx, "userTopContent", vars)
	if err != nil {
		return TopTracksResult{}, err
	}
	items, total := extractUserTopTracks(payload)
	return TopTracksResult{
		Total:  total,
		Limit:  limit,
		Offset: offset,
		Items:  items,
	}, nil
}

func connectTopTimeRange(timeRange string) string {
	switch timeRange {
	case "long_term":
		return "LONG_TERM"
	case "medium_term":
		return "MID_TERM"
	case "short_term":
		return "SHORT_TERM"
	default:
		return timeRange
	}
}

func extractUserTopTracks(payload map[string]any) ([]Item, int) {
	container, ok := getMap(payload, "data", "me", "profile", "topTracks")
	if !ok {
		return nil, 0
	}
	rawItems, _ := container["items"].([]any)
	items := make([]Item, 0, len(rawItems))
	for _, raw := range rawItems {
		wrapper, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		data, ok := wrapper["data"].(map[string]any)
		if !ok {
			continue
		}
		if item, ok := extractItem(data, "track"); ok {
			items = append(items, item)
		}
	}
	total := getInt(container, "totalCount")
	if total == 0 && len(items) > 0 {
		total = len(items)
	}
	return items, total
}

func (c *ConnectClient) recentlyPlayed(ctx context.Context, limit int, after, before int64) (RecentlyPlayedResult, error) {
	if limit <= 0 {
		limit = recentlyPlayedPage
	}
	username, err := c.currentUsername(ctx)
	if err != nil {
		return RecentlyPlayedResult{}, err
	}
	contexts, err := c.recentlyPlayedContexts(ctx, username, limit, after, before)
	if err != nil {
		return RecentlyPlayedResult{}, err
	}
	tracks, err := c.recentlyPlayedTracks(ctx, contexts)
	if err != nil {
		return RecentlyPlayedResult{}, err
	}
	items := make([]RecentlyPlayedItem, 0, len(contexts))
	for _, ctxItem := range contexts {
		track, ok := tracks[ctxItem.LastPlayedTrackURI]
		if !ok {
			id := idFromURI(ctxItem.LastPlayedTrackURI)
			track = Item{
				ID:   id,
				URI:  ctxItem.LastPlayedTrackURI,
				Type: "track",
				URL:  fmt.Sprintf("https://open.spotify.com/track/%s", id),
			}
		}
		items = append(items, RecentlyPlayedItem{
			Track:    track,
			PlayedAt: time.UnixMilli(ctxItem.LastPlayedTime).UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		})
	}
	result := RecentlyPlayedResult{Items: items, Limit: limit}
	if len(contexts) > 0 {
		result.Cursors = &Cursors{
			After:  fmt.Sprint(contexts[0].LastPlayedTime),
			Before: fmt.Sprint(contexts[len(contexts)-1].LastPlayedTime),
		}
	}
	return result, nil
}

func (c *ConnectClient) currentUsername(ctx context.Context) (string, error) {
	payload, err := c.graphQL(ctx, "profileAttributes", map[string]any{})
	if err != nil {
		return "", err
	}
	profile, ok := getMap(payload, "data", "me", "profile")
	if !ok {
		return "", errors.New("missing profile")
	}
	username := getString(profile, "username")
	if username == "" {
		username = idFromURI(getString(profile, "uri"))
	}
	if username == "" || username == "me" {
		return "", errors.New("missing user id")
	}
	return username, nil
}

func (c *ConnectClient) recentlyPlayedContexts(ctx context.Context, username string, limit int, after, before int64) ([]recentlyPlayedContext, error) {
	out := make([]recentlyPlayedContext, 0, limit)
	for offset := 0; len(out) < limit; offset += recentlyPlayedPage {
		page, err := c.recentlyPlayedContextPage(ctx, username, recentlyPlayedPage, offset)
		if err != nil {
			return nil, err
		}
		if len(page.PlayContexts) == 0 {
			break
		}
		for _, item := range page.PlayContexts {
			if item.LastPlayedTrackURI == "" || item.LastPlayedTime == 0 {
				continue
			}
			if before > 0 && item.LastPlayedTime >= before {
				continue
			}
			if after > 0 && item.LastPlayedTime <= after {
				continue
			}
			out = append(out, item)
			if len(out) >= limit {
				break
			}
		}
		oldest := page.PlayContexts[len(page.PlayContexts)-1].LastPlayedTime
		if len(page.PlayContexts) < recentlyPlayedPage || (after > 0 && oldest <= after) {
			break
		}
	}
	return out, nil
}

func (c *ConnectClient) recentlyPlayedContextPage(ctx context.Context, username string, limit, offset int) (recentlyPlayedContextsResponse, error) {
	params := url.Values{}
	params.Set("format", "json")
	params.Set("offset", fmt.Sprint(offset))
	params.Set("limit", fmt.Sprint(limit))
	params.Set("filter", "default,collection-new-episodes")
	requestURL := fmt.Sprintf("%s/user/%s/recently-played?%s", recentlyPlayedBaseURL, url.PathEscape(username), params.Encode())
	var payload recentlyPlayedContextsResponse
	if err := c.connectGet(ctx, requestURL, &payload); err != nil {
		return recentlyPlayedContextsResponse{}, err
	}
	return payload, nil
}

func (c *ConnectClient) recentlyPlayedTracks(ctx context.Context, contexts []recentlyPlayedContext) (map[string]Item, error) {
	seen := map[string]struct{}{}
	uris := make([]string, 0, len(contexts))
	for _, item := range contexts {
		if item.LastPlayedTrackURI == "" {
			continue
		}
		if _, ok := seen[item.LastPlayedTrackURI]; ok {
			continue
		}
		seen[item.LastPlayedTrackURI] = struct{}{}
		uris = append(uris, item.LastPlayedTrackURI)
	}
	if len(uris) == 0 {
		return map[string]Item{}, nil
	}
	payload, err := c.graphQL(ctx, "fetchEntitiesForRecentlyPlayed", map[string]any{"uris": uris})
	if err != nil {
		return nil, err
	}
	return extractRecentlyPlayedTracks(payload), nil
}

func extractRecentlyPlayedTracks(payload map[string]any) map[string]Item {
	out := map[string]Item{}
	data, ok := getMap(payload, "data")
	if !ok {
		return out
	}
	rawItems, _ := data["lookup"].([]any)
	for _, raw := range rawItems {
		wrapper, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		data, ok := wrapper["data"].(map[string]any)
		if !ok {
			continue
		}
		item, ok := extractItem(data, "track")
		if !ok {
			continue
		}
		if uri := getString(wrapper, "_uri"); uri != "" {
			out[uri] = item
		}
		out[item.URI] = item
	}
	return out
}

func (c *ConnectClient) connectGet(ctx context.Context, requestURL string, dest any) error {
	if c == nil || c.session == nil || c.client == nil {
		return errors.New("connect client not initialized")
	}
	auth, err := c.session.auth(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	applyRequestHeaders(req, requestHeaders{
		AccessToken:   auth.AccessToken,
		ClientToken:   auth.ClientToken,
		ClientVersion: auth.ClientVersion,
		Accept:        "application/json",
		Language:      c.language,
		AppPlatform:   defaultSpotifyAppPlatform,
	})
	client := c.searchClient
	if client == nil {
		client = c.client
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiErrorFromResponse(resp)
	}
	if dest == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

package spotify

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

func (c *Client) PlaylistTracks(ctx context.Context, id string, limit, offset int) ([]Item, int, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprint(limit))
	params.Set("offset", fmt.Sprint(offset))
	var raw playlistTracksResponse
	if err := c.get(ctx, "/playlists/"+id+"/items", params, &raw); err != nil {
		return nil, 0, err
	}
	items := make([]Item, 0, len(raw.Items))
	for _, entry := range raw.Items {
		track := entry.track()
		if track.ID == "" {
			continue
		}
		items = append(items, mapTrack(track))
	}
	return items, raw.Total, nil
}

func (c *Client) CreatePlaylist(ctx context.Context, name string, public, collaborative bool) (Item, error) {
	payload := map[string]any{
		"name":          name,
		"public":        public,
		"collaborative": collaborative,
	}
	var raw playlistItem
	if err := c.postJSON(ctx, "/me/playlists", payload, &raw); err != nil {
		return Item{}, err
	}
	return mapPlaylist(raw), nil
}

func (c *Client) AddTracks(ctx context.Context, playlistID string, uris []string) error {
	payload := map[string]any{
		"uris": uris,
	}
	return c.postJSON(ctx, "/playlists/"+playlistID+"/items", payload, nil)
}

func (c *Client) RemoveTracks(ctx context.Context, playlistID string, uris []string) error {
	items := make([]map[string]string, 0, len(uris))
	for _, uri := range uris {
		items = append(items, map[string]string{"uri": uri})
	}
	payload := map[string]any{"items": items}
	return c.send(ctx, http.MethodDelete, "/playlists/"+playlistID+"/items", nil, payload, nil)
}

func (c *Client) FollowPlaylist(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("uris", "spotify:playlist:"+id)
	return c.send(ctx, http.MethodPut, "/me/library", params, nil, nil)
}

func (c *Client) UnfollowPlaylist(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("uris", "spotify:playlist:"+id)
	return c.send(ctx, http.MethodDelete, "/me/library", params, nil, nil)
}

func (c *Client) IsFollowingPlaylist(ctx context.Context, id string) (bool, error) {
	params := url.Values{}
	params.Set("uris", "spotify:playlist:"+id)
	var raw []*bool
	if err := c.get(ctx, "/me/library/contains", params, &raw); err != nil {
		return false, err
	}
	if len(raw) != 1 || raw[0] == nil {
		return false, errors.New("expected one boolean in playlist contains response")
	}
	return *raw[0], nil
}

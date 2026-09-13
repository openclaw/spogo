package spotify

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) LibraryTracks(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return c.libraryTracks(ctx, "/me/tracks", limit, offset)
}

func (c *Client) LibraryAlbums(ctx context.Context, limit, offset int) ([]Item, int, error) {
	return c.libraryTracks(ctx, "/me/albums", limit, offset)
}

func (c *Client) libraryTracks(ctx context.Context, path string, limit, offset int) ([]Item, int, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprint(limit))
	params.Set("offset", fmt.Sprint(offset))
	var raw libraryResponse
	if err := c.get(ctx, path, params, &raw); err != nil {
		return nil, 0, err
	}
	items := make([]Item, 0, len(raw.Items))
	for _, item := range raw.Items {
		if item.Track.ID != "" {
			items = append(items, mapTrack(item.Track))
		}
		if item.Album.ID != "" {
			items = append(items, mapAlbum(item.Album))
		}
	}
	return items, raw.Total, nil
}

func (c *Client) LibraryModify(ctx context.Context, path string, ids []string, method string) error {
	params := url.Values{}
	params.Set("ids", joinComma(ids))
	return c.send(ctx, method, path, params, nil, nil)
}

func (c *Client) FollowArtists(ctx context.Context, ids []string, method string) error {
	params := url.Values{}
	params.Set("type", "artist")
	params.Set("ids", joinComma(ids))
	return c.send(ctx, method, "/me/following", params, nil, nil)
}

func (c *Client) Playlists(ctx context.Context, limit, offset int) ([]Item, int, error) {
	params := url.Values{}
	params.Set("limit", fmt.Sprint(limit))
	params.Set("offset", fmt.Sprint(offset))
	var raw playlistListResponse
	if err := c.get(ctx, "/me/playlists", params, &raw); err != nil {
		return nil, 0, err
	}
	items := make([]Item, 0, len(raw.Items))
	for _, item := range raw.Items {
		items = append(items, mapPlaylist(item))
	}
	return items, raw.Total, nil
}

func (c *Client) FollowedArtists(ctx context.Context, limit int, after string) ([]Item, int, string, error) {
	params := url.Values{}
	params.Set("type", "artist")
	params.Set("limit", fmt.Sprint(limit))
	if after != "" {
		params.Set("after", after)
	}
	var raw followedArtistsResponse
	if err := c.get(ctx, "/me/following", params, &raw); err != nil {
		return nil, 0, "", err
	}
	items := make([]Item, 0, len(raw.Artists.Items))
	for _, artist := range raw.Artists.Items {
		items = append(items, mapArtist(artist))
	}
	nextAfter := ""
	if len(raw.Artists.Items) > 0 {
		nextAfter = raw.Artists.Items[len(raw.Artists.Items)-1].ID
	}
	return items, raw.Artists.Total, nextAfter, nil
}

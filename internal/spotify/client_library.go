package spotify

import (
	"context"
	"errors"
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

// LibraryModify saves or removes tracks and albums. Spotify replaced the
// entity specific /me/tracks and /me/albums endpoints with /me/library in the
// February 2026 Web API change; the new endpoint takes URIs, not ids.
func (c *Client) LibraryModify(ctx context.Context, kind string, ids []string, method string) error {
	uris := make([]string, 0, len(ids))
	for _, id := range ids {
		uris = append(uris, "spotify:"+kind+":"+id)
	}
	return c.modifyLibrary(ctx, uris, method)
}

func (c *Client) FollowArtists(ctx context.Context, ids []string, method string) error {
	return c.LibraryModify(ctx, "artist", ids, method)
}

func (c *Client) modifyLibrary(ctx context.Context, uris []string, method string) error {
	if len(uris) == 0 {
		return errors.New("no library items provided")
	}
	// The replacement endpoint accepts fewer items than the old typed endpoints.
	const batchSize = 40
	for start := 0; start < len(uris); start += batchSize {
		end := min(start+batchSize, len(uris))
		params := url.Values{}
		params.Set("uris", joinComma(uris[start:end]))
		if err := c.send(ctx, method, "/me/library", params, nil, nil); err != nil {
			return err
		}
	}
	return nil
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

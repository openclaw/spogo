package spotify

import (
	"context"
	"fmt"
)

func (c *ConnectClient) playlists(ctx context.Context, limit, offset int) ([]Item, int, error) {
	payload, err := c.graphQL(ctx, "libraryV3", libraryV3Variables("Playlists", normalizeLibraryLimit(limit), offset))
	if err != nil {
		return nil, 0, err
	}
	items, total := extractLibraryV3Items(payload, "playlist")
	return items, total, nil
}

func (c *ConnectClient) playlistTracks(ctx context.Context, id string, limit, offset int) ([]Item, int, error) {
	payload, err := c.graphQL(ctx, "fetchPlaylist", playlistTrackVariables(id, normalizePlaylistTrackLimit(limit), offset))
	if err != nil {
		return nil, 0, err
	}
	items, total := extractPlaylistContentItems(payload, "track")
	return items, total, nil
}

func (c *ConnectClient) libraryTracks(ctx context.Context, limit, offset int) ([]Item, int, error) {
	vars := map[string]any{
		"uri":    "spotify:collection:tracks",
		"offset": offset,
		"limit":  normalizeLibraryLimit(limit),
	}
	payload, err := c.graphQL(ctx, "fetchLibraryTracks", vars)
	if err != nil {
		return nil, 0, err
	}
	return extractFetchLibraryTracks(payload)
}

func (c *ConnectClient) libraryAlbums(ctx context.Context, limit, offset int) ([]Item, int, error) {
	payload, err := c.graphQL(ctx, "libraryV3", libraryV3Variables("Albums", normalizeLibraryLimit(limit), offset))
	if err != nil {
		return nil, 0, err
	}
	items, total := extractLibraryV3Items(payload, "album")
	return items, total, nil
}

func (c *ConnectClient) followedArtists(ctx context.Context, limit int, after string) ([]Item, int, string, error) {
	limit = normalizeLibraryLimit(limit)
	offset := 0
	if after != "" {
		var err error
		offset, err = c.followedArtistOffset(ctx, after)
		if err != nil {
			return nil, 0, "", err
		}
	}
	payload, err := c.graphQL(ctx, "libraryV3", libraryV3Variables("Artists", limit, offset))
	if err != nil {
		return nil, 0, "", err
	}
	items, total := extractLibraryV3Items(payload, "artist")
	next := ""
	if len(items) > 0 && offset+len(items) < total {
		next = items[len(items)-1].ID
	}
	return items, total, next, nil
}

func (c *ConnectClient) followedArtistOffset(ctx context.Context, after string) (int, error) {
	const pageSize = 50
	for offset := 0; ; offset += pageSize {
		payload, err := c.graphQL(ctx, "libraryV3", libraryV3Variables("Artists", pageSize, offset))
		if err != nil {
			return 0, err
		}
		items, total := extractLibraryV3Items(payload, "artist")
		for index, item := range items {
			if item.ID == after || item.URI == after {
				return offset + index + 1, nil
			}
		}
		if len(items) == 0 || offset+len(items) >= total {
			return 0, fmt.Errorf("followed artist cursor %q not found", after)
		}
	}
}

func normalizeLibraryLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	return limit
}

func normalizePlaylistTrackLimit(limit int) int {
	if limit <= 0 {
		return 25
	}
	return limit
}

func libraryV3Variables(filter string, limit, offset int) map[string]any {
	return map[string]any{
		"filters":                      []any{filter},
		"order":                        nil,
		"textFilter":                   "",
		"features":                     []any{},
		"limit":                        limit,
		"offset":                       offset,
		"flatten":                      false,
		"expandedFolders":              []any{},
		"folderUri":                    nil,
		"includeFoldersWhenFlattening": true,
		"withCuration":                 false,
	}
}

func playlistTrackVariables(id string, limit, offset int) map[string]any {
	return map[string]any{
		"uri":                       "spotify:playlist:" + id,
		"offset":                    offset,
		"limit":                     limit,
		"enableWatchFeedEntrypoint": false,
	}
}

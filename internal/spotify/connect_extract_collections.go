package spotify

import "fmt"

// extractLibraryV3Items navigates the specific libraryV3 response path
// data.me.libraryV3.items[i].item.data to extract items of the given kind.
// Using a targeted path avoids the duplicates and fake sort-category entries
// that a full recursive walk would produce.
func extractLibraryV3Items(payload map[string]any, kind string) ([]Item, int) {
	lib, ok := getMap(payload, "data", "me", "libraryV3")
	if !ok {
		return nil, 0
	}
	items := dedupeCollectionItems(extractWrappedCollectionItems(lib, "item", kind))
	return items, collectionTotal(lib, items)
}

// extractFetchLibraryTracks navigates the fetchLibraryTracks response path
// data.me.library.tracks.items[i].track.data to extract track items.
// The track URI lives at items[i].track._uri (not inside .data), so we
// inject it into the data map before passing it to extractItem.
func extractFetchLibraryTracks(payload map[string]any) ([]Item, int, error) {
	tracks, ok := getMap(payload, "data", "me", "library", "tracks")
	if !ok {
		return nil, 0, fmt.Errorf("fetchLibraryTracks payload missing data.me.library.tracks")
	}
	rawItemsValue, ok := tracks["items"]
	if !ok {
		return nil, 0, fmt.Errorf("fetchLibraryTracks payload missing data.me.library.tracks.items")
	}
	rawItems, ok := rawItemsValue.([]any)
	if !ok {
		return nil, 0, fmt.Errorf("fetchLibraryTracks payload has invalid data.me.library.tracks.items")
	}
	items := make([]Item, 0, len(rawItems))
	for _, raw := range rawItems {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		wrapper, ok := m["track"].(map[string]any)
		if !ok {
			continue
		}
		dataM, ok := wrapper["data"].(map[string]any)
		if !ok {
			continue
		}
		// _uri is on the wrapper, not inside data
		if uri, ok := wrapper["_uri"].(string); ok && getString(dataM, "uri") == "" {
			dataM["uri"] = uri
		}
		item, ok := extractItem(dataM, "track")
		if !ok {
			continue
		}
		items = append(items, item)
	}
	items = dedupeCollectionItems(items)
	return items, collectionTotal(tracks, items), nil
}

func extractPlaylistContentItems(payload map[string]any, kind string) ([]Item, int) {
	content, ok := getMap(payload, "data", "playlistV2", "content")
	if !ok {
		return nil, 0
	}
	// A playlist is an ordered sequence; repeated URIs are distinct positions.
	items := extractWrappedCollectionItems(content, "itemV2", kind)
	return items, collectionTotal(content, items)
}

func extractWrappedCollectionItems(container map[string]any, wrapperKey, kind string) []Item {
	rawItems, _ := container["items"].([]any)
	items := make([]Item, 0, len(rawItems))
	for _, raw := range rawItems {
		data, ok := getMap(raw, wrapperKey, "data")
		if !ok {
			continue
		}
		if item, ok := extractItem(data, kind); ok {
			items = append(items, item)
		}
	}
	return items
}

func dedupeCollectionItems(items []Item) []Item {
	seen := make(map[string]struct{}, len(items))
	unique := items[:0]
	for _, item := range items {
		if _, exists := seen[item.URI]; exists {
			continue
		}
		seen[item.URI] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}

func collectionTotal(container map[string]any, items []Item) int {
	if total := getInt(container, "totalCount"); total != 0 {
		return total
	}
	return len(items)
}

package spotify

import "strings"

func extractSearchItems(payload map[string]any, kind string) ([]Item, int) {
	for _, path := range searchPaths(kind) {
		if container, ok := getMap(payload, path...); ok {
			items := extractItemsFromContainer(container, kind)
			total := getInt(container, "totalCount")
			if total == 0 {
				total = len(items)
			}
			return items, total
		}
	}
	items := collectItemsByKind(payload, kind)
	return items, len(items)
}

func extractItemFromPayload(payload map[string]any, kind string) (Item, bool) {
	return extractRequestedItemFromPayload(payload, kind, "")
}

func extractRequestedItemFromPayload(payload map[string]any, kind, requestedURI string) (Item, bool) {
	for _, key := range itemPayloadKeys(kind) {
		if entity, ok := getMap(payload, "data", key); ok {
			if item, ok := extractRequestedEntity(entity, kind, requestedURI); ok {
				return item, true
			}
		}
	}
	if requestedURI != "" {
		var matched map[string]any
		walkMap(payload, func(candidate map[string]any) {
			if matched == nil && entityMatchesURI(candidate, kind, requestedURI) {
				matched = candidate
			}
		})
		if matched != nil {
			if item, ok := extractRequestedEntity(matched, kind, requestedURI); ok {
				return item, true
			}
		}
	}
	items := collectItemsByKind(payload, kind)
	for _, item := range items {
		if requestedURI == "" || item.URI == requestedURI {
			return item, true
		}
	}
	return Item{}, false
}

func itemPayloadKeys(kind string) []string {
	switch kind {
	case "track":
		return []string{"trackUnion", "track"}
	case "album":
		return []string{"albumUnion", "album"}
	case "artist":
		return []string{"artistUnion", "artist"}
	case "playlist":
		return []string{"playlistV2", "playlist"}
	case "show":
		return []string{"podcastUnionV2", "podcastUnion", "showUnion", "podcast", "show"}
	case "episode":
		return []string{"episodeUnionV2", "episodeUnion", "episodeOrChapter", "episode"}
	default:
		return nil
	}
}

func extractRequestedEntity(entity map[string]any, kind, requestedURI string) (Item, bool) {
	if requestedURI != "" && !entityMatchesURI(entity, kind, requestedURI) {
		return Item{}, false
	}
	if kind == "artist" {
		if profile, ok := getMap(entity, "profile"); ok {
			if name := getString(profile, "name"); name != "" {
				named := make(map[string]any, len(entity)+1)
				for key, value := range entity {
					named[key] = value
				}
				named["name"] = name
				entity = named
			}
		}
	}
	item, ok := extractItem(entity, kind)
	if !ok {
		return Item{}, false
	}
	if kind == "artist" {
		item.Genres = extractEntityGenres(entity)
	}
	return item, true
}

func entityMatchesURI(entity map[string]any, kind, requestedURI string) bool {
	if uri := getString(entity, "uri"); uri != "" {
		return uri == requestedURI
	}
	return getString(entity, "id") == strings.TrimPrefix(requestedURI, "spotify:"+kind+":")
}

func extractEntityGenres(entity map[string]any) []string {
	genres := extractGenreNames(entity["genres"])
	if len(genres) == 0 {
		if profile, ok := getMap(entity, "profile"); ok {
			genres = extractGenreNames(profile["genres"])
		}
	}
	return dedupeStrings(genres)
}

func extractGenreNames(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		genres := make([]string, 0, len(typed))
		for _, raw := range typed {
			switch genre := raw.(type) {
			case string:
				genres = append(genres, genre)
			case map[string]any:
				if name := getString(genre, "name"); name != "" {
					genres = append(genres, name)
				}
			}
		}
		return genres
	case map[string]any:
		for _, key := range []string{"items", "nodes"} {
			if genres := extractGenreNames(typed[key]); len(genres) > 0 {
				return genres
			}
		}
	}
	return nil
}

func searchPaths(kind string) [][]string {
	switch kind {
	case "track":
		return [][]string{{"data", "searchV2", "tracksV2"}}
	case "album":
		return [][]string{{"data", "searchV2", "albumsV2"}, {"data", "searchV2", "albums"}}
	case "artist":
		return [][]string{{"data", "searchV2", "artists"}}
	case "playlist":
		return [][]string{{"data", "searchV2", "playlists"}}
	case "show":
		return [][]string{{"data", "searchV2", "podcasts"}, {"data", "searchV2", "shows"}}
	case "episode":
		return [][]string{{"data", "searchV2", "episodes"}}
	default:
		return nil
	}
}

func extractItemsFromContainer(container map[string]any, kind string) []Item {
	itemsRaw, ok := container["items"].([]any)
	if !ok {
		return collectItemsByKind(container, kind)
	}
	items := make([]Item, 0, len(itemsRaw))
	for _, raw := range itemsRaw {
		if item, ok := extractItem(raw, kind); ok {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return collectItemsByKind(container, kind)
	}
	return items
}

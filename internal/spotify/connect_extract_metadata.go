package spotify

import "strings"

func extractArtistNames(value any) []string {
	artists := []string{}
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	if list, ok := m["artists"].([]any); ok {
		appendArtistNames(&artists, list)
	}
	if group, ok := m["artists"].(map[string]any); ok {
		if list, ok := group["items"].([]any); ok {
			appendArtistNames(&artists, list)
		}
		if list, ok := group["nodes"].([]any); ok {
			appendArtistNames(&artists, list)
		}
		if list, ok := group["edges"].([]any); ok {
			appendArtistNames(&artists, list)
		}
	}
	if group, ok := m["firstArtist"].(map[string]any); ok {
		if list, ok := group["items"].([]any); ok {
			appendArtistNames(&artists, list)
		}
		if list, ok := group["nodes"].([]any); ok {
			appendArtistNames(&artists, list)
		}
		if list, ok := group["edges"].([]any); ok {
			appendArtistNames(&artists, list)
		}
	}
	if group, ok := m["otherArtists"].(map[string]any); ok {
		if list, ok := group["items"].([]any); ok {
			appendArtistNames(&artists, list)
		}
		if list, ok := group["nodes"].([]any); ok {
			appendArtistNames(&artists, list)
		}
		if list, ok := group["edges"].([]any); ok {
			appendArtistNames(&artists, list)
		}
	}
	if len(artists) == 0 {
		if name := getString(m, "artistName"); name != "" {
			artists = append(artists, name)
		}
	}
	return dedupeStrings(artists)
}

func appendArtistNames(artists *[]string, entries []any) {
	for _, entry := range entries {
		if name := artistNameFromValue(entry); name != "" {
			*artists = append(*artists, name)
		}
	}
}

func findFirstArtistNames(value any) []string {
	var artists []string
	walkMap(value, func(m map[string]any) {
		if len(artists) > 0 {
			return
		}
		if found := extractArtistNames(m); len(found) > 0 {
			artists = found
		}
	})
	return artists
}

func artistNameFromValue(value any) string {
	m, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	if profile, ok := m["profile"].(map[string]any); ok {
		if name := getString(profile, "name"); name != "" {
			return name
		}
	}
	if node, ok := m["node"].(map[string]any); ok {
		if name := artistNameFromValue(node); name != "" {
			return name
		}
	}
	if artist, ok := m["artist"].(map[string]any); ok {
		if name := artistNameFromValue(artist); name != "" {
			return name
		}
	}
	name := getString(m, "name")
	if name == "" {
		return ""
	}
	if len(m) == 1 || isArtistMap(m) || getString(m, "id") != "" {
		return name
	}
	return ""
}

func isArtistMap(m map[string]any) bool {
	if uri := getString(m, "uri"); strings.HasPrefix(uri, "spotify:artist:") {
		return true
	}
	if typ := getString(m, "type"); typ == "artist" {
		return true
	}
	if _, ok := m["profile"]; ok {
		return true
	}
	return false
}

func extractAlbumName(value any) string {
	var album string
	walkMap(value, func(m map[string]any) {
		if album != "" {
			return
		}
		if inner, ok := m["album"].(map[string]any); ok {
			if name := getString(inner, "name"); name != "" {
				album = name
			}
		}
		if inner, ok := m["albumOfTrack"].(map[string]any); ok {
			if name := getString(inner, "name"); name != "" {
				album = name
			}
		}
	})
	return album
}

func extractOwnerName(value any) string {
	if entity, ok := value.(map[string]any); ok {
		for _, path := range [][]string{{"ownerV2", "data"}, {"owner"}, {"user", "data"}, {"user"}} {
			if owner, ok := getMap(entity, path...); ok {
				if name := getString(owner, "name"); name != "" {
					return name
				}
				if name := getString(owner, "display_name"); name != "" {
					return name
				}
			}
		}
	}
	var owner string
	walkMap(value, func(m map[string]any) {
		if owner != "" {
			return
		}
		if inner, ok := m["owner"].(map[string]any); ok {
			if name := getString(inner, "name"); name != "" {
				owner = name
			}
		}
		if inner, ok := m["user"].(map[string]any); ok {
			if name := getString(inner, "name"); name != "" {
				owner = name
			}
		}
	})
	return owner
}

func extractShowName(entity map[string]any) string {
	for _, path := range [][]string{{"podcastV2", "data"}, {"podcast", "data"}, {"podcast"}, {"show"}} {
		if show, ok := getMap(entity, path...); ok {
			if name := getString(show, "name"); name != "" {
				return name
			}
		}
	}
	return ""
}

func walkMap(value any, fn func(map[string]any)) {
	switch typed := value.(type) {
	case map[string]any:
		fn(typed)
		for _, child := range typed {
			walkMap(child, fn)
		}
	case []any:
		for _, child := range typed {
			walkMap(child, fn)
		}
	}
}

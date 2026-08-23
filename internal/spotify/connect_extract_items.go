package spotify

import (
	"fmt"
	"strconv"
	"strings"
)

func collectItemsByKind(value any, kind string) []Item {
	items := []Item{}
	visitItems(value, kind, &items)
	return items
}

func visitItems(value any, kind string, items *[]Item) {
	switch typed := value.(type) {
	case map[string]any:
		if item, ok := extractItem(typed, kind); ok {
			*items = append(*items, item)
		}
		for _, child := range typed {
			visitItems(child, kind, items)
		}
	case []any:
		for _, child := range typed {
			visitItems(child, kind, items)
		}
	}
}

func extractItem(value any, kind string) (Item, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return Item{}, false
	}
	if data, ok := m["data"].(map[string]any); ok {
		if uri := getString(m, "_uri"); uri != "" && getString(data, "uri") == "" {
			data = copyEntityWithURI(data, uri)
		}
		m = data
	}
	if wrapped, ok := getMap(m, "item", "data"); ok {
		m = wrapped
	}
	if kind == "track" {
		if inner, ok := m["track"].(map[string]any); ok {
			m = inner
		}
	}
	uri := getString(m, "uri")
	if uri == "" && kind != "" {
		if id := getString(m, "id"); id != "" {
			uri = "spotify:" + kind + ":" + id
		}
	}
	if uri == "" {
		if inner := findFirstURI(m, kind); inner != "" {
			uri = inner
		}
	}
	if uri == "" {
		return Item{}, false
	}
	if kind != "" && !strings.HasPrefix(uri, "spotify:"+kind+":") {
		return Item{}, false
	}
	name := getString(m, "name")
	if name == "" {
		name = getString(m, "title")
	}
	if name == "" {
		if profile, ok := getMap(m, "profile"); ok {
			name = getString(profile, "name")
		}
	}
	if name == "" {
		name = findFirstName(m)
	}
	item := Item{
		URI:  uri,
		ID:   idFromURI(uri),
		Name: name,
		Type: typeFromURI(uri),
	}
	item.URL = fmt.Sprintf("https://open.spotify.com/%s/%s", item.Type, item.ID)
	item.Artists = extractArtistNames(m)
	if len(item.Artists) == 0 && item.Type == "track" {
		item.Artists = findFirstArtistNames(m)
	}
	if item.Type == "track" {
		if album := extractAlbumName(m); album != "" {
			item.Album = album
		}
	}
	if item.Type == "episode" {
		item.Show = extractShowName(m)
	}
	if _, ok := m["explicit"]; ok {
		item.Explicit = getBool(m, "explicit")
		item.ExplicitKnown = true
	}
	if rating, ok := m["contentRating"].(map[string]any); ok {
		label := strings.ToUpper(getString(rating, "label"))
		if label != "" {
			item.Explicit = label == "EXPLICIT"
			item.ExplicitKnown = true
		}
	}
	item.DurationMS = getInt(m, "duration_ms")
	if item.DurationMS == 0 {
		item.DurationMS = getInt(m, "durationMs")
	}
	if item.DurationMS == 0 {
		item.DurationMS = getNestedInt(m, "duration", "totalMilliseconds")
	}
	if item.DurationMS == 0 {
		item.DurationMS = getNestedInt(m, "trackDuration", "totalMilliseconds")
	}
	item.Owner = extractOwnerName(m)
	item.TotalTracks = firstPositiveInt(
		getInt(m, "totalTracks"),
		getInt(m, "total_tracks"),
		getNestedInt(m, "tracks", "totalCount"),
		getNestedInt(m, "tracks", "total"),
		getNestedInt(m, "tracksV2", "totalCount"),
		getNestedInt(m, "content", "totalCount"),
		getInt(m, "total"),
	)
	item.ReleaseDate = extractReleaseDate(m)
	item.Description = getString(m, "description")
	item.IsPlayable = getBool(m, "isPlayable")
	if !item.IsPlayable {
		item.IsPlayable = getNestedBool(m, "playability", "playable")
	}
	item.Publisher = getString(m, "publisher")
	if item.Publisher == "" {
		if publisher, ok := getMap(m, "publisher"); ok {
			item.Publisher = getString(publisher, "name")
		}
	}
	item.TotalEpisodes = firstPositiveInt(
		getInt(m, "totalEpisodes"),
		getInt(m, "total_episodes"),
		getNestedInt(m, "episodes", "totalCount"),
		getNestedInt(m, "episodesV2", "totalCount"),
	)
	item.Followers = firstPositiveInt(
		getInt(m, "followers"),
		getNestedInt(m, "stats", "followers"),
		getNestedInt(m, "followers", "total"),
	)
	return item, true
}

func copyEntityWithURI(entity map[string]any, uri string) map[string]any {
	copy := make(map[string]any, len(entity)+1)
	for key, value := range entity {
		copy[key] = value
	}
	copy["uri"] = uri
	return copy
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func extractReleaseDate(entity map[string]any) string {
	if date := getString(entity, "release_date"); date != "" {
		return date
	}
	if date := getString(entity, "releaseDate"); date != "" {
		return date
	}
	for _, key := range []string{"releaseDate", "date"} {
		date, ok := getMap(entity, key)
		if !ok {
			continue
		}
		if value := getString(date, "isoString"); value != "" {
			value, _, _ = strings.Cut(value, "T")
			return value
		}
		if year := getInt(date, "year"); year > 0 {
			return strconv.Itoa(year)
		}
	}
	return ""
}

func idFromURI(uri string) string {
	parts := strings.Split(uri, ":")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return uri
}

func typeFromURI(uri string) string {
	parts := strings.Split(uri, ":")
	if len(parts) >= 3 {
		return parts[len(parts)-2]
	}
	return ""
}

func findFirstURI(value any, kind string) string {
	switch typed := value.(type) {
	case map[string]any:
		if uri, ok := typed["uri"].(string); ok {
			if kind == "" || strings.HasPrefix(uri, "spotify:"+kind+":") {
				return uri
			}
		}
		for _, child := range typed {
			if uri := findFirstURI(child, kind); uri != "" {
				return uri
			}
		}
	case []any:
		for _, child := range typed {
			if uri := findFirstURI(child, kind); uri != "" {
				return uri
			}
		}
	}
	return ""
}

func findFirstName(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		if name, ok := typed["name"].(string); ok {
			return name
		}
		if title, ok := typed["title"].(string); ok {
			return title
		}
		for _, child := range typed {
			if name := findFirstName(child); name != "" {
				return name
			}
		}
	case []any:
		for _, child := range typed {
			if name := findFirstName(child); name != "" {
				return name
			}
		}
	}
	return ""
}

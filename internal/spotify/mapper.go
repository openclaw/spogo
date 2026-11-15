package spotify

import "encoding/json"

func mapSearchItem(kind string, raw json.RawMessage) (Item, error) {
	switch kind {
	case "track":
		var t trackItem
		if err := json.Unmarshal(raw, &t); err != nil {
			return Item{}, err
		}
		return mapTrack(t), nil
	case "album":
		var a albumItem
		if err := json.Unmarshal(raw, &a); err != nil {
			return Item{}, err
		}
		return mapAlbum(a), nil
	case "artist":
		var a artistItem
		if err := json.Unmarshal(raw, &a); err != nil {
			return Item{}, err
		}
		return mapArtist(a), nil
	case "playlist":
		var p playlistItem
		if err := json.Unmarshal(raw, &p); err != nil {
			return Item{}, err
		}
		return mapPlaylist(p), nil
	case "show":
		var s showItem
		if err := json.Unmarshal(raw, &s); err != nil {
			return Item{}, err
		}
		return mapShow(s), nil
	case "episode":
		var e episodeItem
		if err := json.Unmarshal(raw, &e); err != nil {
			return Item{}, err
		}
		return mapEpisode(e), nil
	default:
		return Item{}, ErrUnsupportedType
	}
}

func mapTrack(t trackItem) Item {
	return Item{
		ID:         t.ID,
		URI:        t.URI,
		Name:       t.Name,
		Type:       "track",
		URL:        externalURL(t.ExternalURLs),
		Artists:    artistNames(t.Artists),
		Album:      t.Album.Name,
		DurationMS: t.DurationMS,
		Explicit:   t.Explicit,
		IsPlayable: t.IsPlayable,
	}
}

func mapAlbum(a albumItem) Item {
	return Item{
		ID:          a.ID,
		URI:         a.URI,
		Name:        a.Name,
		Type:        "album",
		URL:         externalURL(a.ExternalURLs),
		Artists:     artistNames(a.Artists),
		ReleaseDate: a.ReleaseDate,
		TotalTracks: a.TotalTracks,
	}
}

func mapArtist(a artistItem) Item {
	return Item{
		ID:        a.ID,
		URI:       a.URI,
		Name:      a.Name,
		Type:      "artist",
		URL:       externalURL(a.ExternalURLs),
		Followers: a.Followers.Total,
		Genres:    a.Genres,
	}
}

// playlistTotal prefers the post-2026 "items" container and falls back to
// "tracks", which Extended Quota Mode apps still receive.
func playlistTotal(p playlistItem) int {
	if p.Items.Total != 0 {
		return p.Items.Total
	}
	return p.Tracks.Total
}

func mapPlaylist(p playlistItem) Item {
	return Item{
		ID:          p.ID,
		URI:         p.URI,
		Name:        p.Name,
		Type:        "playlist",
		URL:         externalURL(p.ExternalURLs),
		Owner:       p.Owner.DisplayName,
		TotalTracks: playlistTotal(p),
		Description: p.Description,
	}
}

func mapShow(s showItem) Item {
	return Item{
		ID:            s.ID,
		URI:           s.URI,
		Name:          s.Name,
		Type:          "show",
		URL:           externalURL(s.ExternalURLs),
		Description:   s.Description,
		Publisher:     s.Publisher,
		TotalEpisodes: s.TotalEpisodes,
	}
}

func mapEpisode(e episodeItem) Item {
	return Item{
		ID:          e.ID,
		URI:         e.URI,
		Name:        e.Name,
		Type:        "episode",
		URL:         externalURL(e.ExternalURLs),
		Description: e.Description,
		DurationMS:  e.DurationMS,
	}
}

func mapDevice(d deviceItem) Device {
	return Device(d)
}

func artistNames(artists []artistRef) []string {
	ret := make([]string, 0, len(artists))
	for _, a := range artists {
		if a.Name == "" {
			continue
		}
		ret = append(ret, a.Name)
	}
	return ret
}

func externalURL(urls map[string]string) string {
	if urls == nil {
		return ""
	}
	if url := urls["spotify"]; url != "" {
		return url
	}
	for _, url := range urls {
		return url
	}
	return ""
}

func itemNeedsTrackMetadata(item *Item) bool {
	if item == nil || item.ID == "" {
		return false
	}
	if item.Type != "" && item.Type != "track" {
		return false
	}
	return item.Name == "" || len(item.Artists) == 0 || item.Album == ""
}

func mergeItemMetadata(dst *Item, src Item) {
	if dst == nil {
		return
	}
	if dst.ID == "" {
		dst.ID = src.ID
	}
	if dst.URI == "" {
		dst.URI = src.URI
	}
	if dst.Name == "" {
		dst.Name = src.Name
	}
	if dst.Type == "" {
		dst.Type = src.Type
	}
	if dst.URL == "" {
		dst.URL = src.URL
	}
	if len(dst.Artists) == 0 {
		dst.Artists = src.Artists
	}
	if dst.Album == "" {
		dst.Album = src.Album
	}
	if dst.DurationMS == 0 {
		dst.DurationMS = src.DurationMS
	}
	if !dst.ExplicitKnown && (src.ExplicitKnown || src.Explicit) {
		dst.Explicit = src.Explicit
		dst.ExplicitKnown = src.ExplicitKnown
	}
	if !dst.IsPlayable {
		dst.IsPlayable = src.IsPlayable
	}
}

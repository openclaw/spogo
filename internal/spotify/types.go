package spotify

import "encoding/json"

type Item struct {
	ID            string   `json:"id"`
	URI           string   `json:"uri"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	URL           string   `json:"url"`
	Artists       []string `json:"artists,omitempty"`
	Album         string   `json:"album,omitempty"`
	Owner         string   `json:"owner,omitempty"`
	DurationMS    int      `json:"duration_ms,omitempty"`
	Explicit      bool     `json:"-"`
	ExplicitKnown bool     `json:"-"`
	TotalTracks   int      `json:"total_tracks,omitempty"`
	ReleaseDate   string   `json:"release_date,omitempty"`
	Description   string   `json:"description,omitempty"`
	TotalItems    int      `json:"total_items,omitempty"`
	Followers     int      `json:"followers,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	IsPlayable    bool     `json:"is_playable,omitempty"`
	Publisher     string   `json:"publisher,omitempty"`
	TotalEpisodes int      `json:"total_episodes,omitempty"`
}

func (i Item) MarshalJSON() ([]byte, error) {
	type itemAlias Item
	var explicit *bool
	if i.Explicit || i.ExplicitKnown {
		explicit = &i.Explicit
	}
	return json.Marshal(struct {
		itemAlias
		Explicit *bool `json:"explicit,omitempty"`
	}{
		itemAlias: itemAlias(i),
		Explicit:  explicit,
	})
}

type SearchResult struct {
	Type   string `json:"type"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Total  int    `json:"total"`
	Items  []Item `json:"items"`
}

type PlaybackStatus struct {
	IsPlaying  bool   `json:"is_playing"`
	ProgressMS int    `json:"progress_ms"`
	Item       *Item  `json:"item,omitempty"`
	Device     Device `json:"device"`
	Shuffle    bool   `json:"shuffle"`
	Repeat     string `json:"repeat"`
}

type Device struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Volume     int    `json:"volume_percent"`
	Active     bool   `json:"is_active"`
	Restricted bool   `json:"is_restricted"`
}

type Queue struct {
	CurrentlyPlaying *Item  `json:"currently_playing,omitempty"`
	Queue            []Item `json:"queue"`
}

type TopTracksResult struct {
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
	Items  []Item `json:"items"`
}

type RecentlyPlayedResult struct {
	Items   []RecentlyPlayedItem `json:"items"`
	Cursors *Cursors             `json:"cursors,omitempty"`
	Next    string               `json:"next,omitempty"`
	Limit   int                  `json:"limit"`
}

type RecentlyPlayedItem struct {
	Track    Item   `json:"track"`
	PlayedAt string `json:"played_at"`
}

type Cursors struct {
	After  string `json:"after"`
	Before string `json:"before"`
}

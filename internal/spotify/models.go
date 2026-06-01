package spotify

import "encoding/json"

type image struct {
	URL string `json:"url"`
}

type artistItem struct {
	ID        string   `json:"id"`
	URI       string   `json:"uri"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Genres    []string `json:"genres"`
	Followers struct {
		Total int `json:"total"`
	} `json:"followers"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type albumItem struct {
	ID           string            `json:"id"`
	URI          string            `json:"uri"`
	Name         string            `json:"name"`
	Type         string            `json:"album_type"`
	ReleaseDate  string            `json:"release_date"`
	TotalTracks  int               `json:"total_tracks"`
	Artists      []artistRef       `json:"artists"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type artistRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type trackItem struct {
	ID           string            `json:"id"`
	URI          string            `json:"uri"`
	Name         string            `json:"name"`
	DurationMS   int               `json:"duration_ms"`
	Explicit     bool              `json:"explicit"`
	IsPlayable   bool              `json:"is_playable"`
	Album        albumRef          `json:"album"`
	Artists      []artistRef       `json:"artists"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type albumRef struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	URI    string  `json:"uri"`
	Images []image `json:"images"`
}

type playlistItem struct {
	ID          string `json:"id"`
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Owner       struct {
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	Tracks struct {
		Total int `json:"total"`
	} `json:"tracks"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type showItem struct {
	ID            string            `json:"id"`
	URI           string            `json:"uri"`
	Name          string            `json:"name"`
	Publisher     string            `json:"publisher"`
	Description   string            `json:"description"`
	TotalEpisodes int               `json:"total_episodes"`
	ExternalURLs  map[string]string `json:"external_urls"`
}

type episodeItem struct {
	ID           string            `json:"id"`
	URI          string            `json:"uri"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	DurationMS   int               `json:"duration_ms"`
	ExternalURLs map[string]string `json:"external_urls"`
}

type searchContainer struct {
	Items  []json.RawMessage `json:"items"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Total  int               `json:"total"`
}

type playbackResponse struct {
	IsPlaying    bool       `json:"is_playing"`
	ProgressMS   int        `json:"progress_ms"`
	ShuffleState bool       `json:"shuffle_state"`
	RepeatState  string     `json:"repeat_state"`
	Device       deviceItem `json:"device"`
	Item         trackItem  `json:"item"`
}

type deviceItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Volume     int    `json:"volume_percent"`
	Active     bool   `json:"is_active"`
	Restricted bool   `json:"is_restricted"`
}

type deviceResponse struct {
	Devices []deviceItem `json:"devices"`
}

type queueResponse struct {
	CurrentlyPlaying trackItem   `json:"currently_playing"`
	Queue            []trackItem `json:"queue"`
}

type libraryResponse struct {
	Items []struct {
		Track trackItem `json:"track"`
		Album albumItem `json:"album"`
	} `json:"items"`
	Total int `json:"total"`
}

type playlistListResponse struct {
	Items []playlistItem `json:"items"`
	Total int            `json:"total"`
}

type playlistTracksResponse struct {
	Items []struct {
		Track trackItem `json:"track"`
	} `json:"items"`
	Total int `json:"total"`
}

type userProfile struct {
	ID string `json:"id"`
}

type followedArtistsResponse struct {
	Artists artistsContainer `json:"artists"`
}

type artistsContainer struct {
	Items  []artistItem `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type artistTopTracksResponse struct {
	Tracks []trackItem `json:"tracks"`
}

type topTracksResponse struct {
	Items  []trackItem `json:"items"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type recentlyPlayedResponse struct {
	Items []struct {
		Track    trackItem `json:"track"`
		PlayedAt string    `json:"played_at"`
	} `json:"items"`
	Cursors *cursorsItem `json:"cursors"`
	Next    string       `json:"next"`
	Limit   int          `json:"limit"`
}

type cursorsItem struct {
	After  string `json:"after"`
	Before string `json:"before"`
}

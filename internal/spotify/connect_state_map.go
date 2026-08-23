package spotify

import (
	"sort"
	"strconv"
	"strings"
)

func mapDevices(state connectState) []Device {
	if state.devices == nil {
		return nil
	}
	devices := make([]Device, 0, len(state.devices))
	for id, raw := range state.devices {
		deviceMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name := getString(deviceMap, "name")
		if name == "" {
			name = getString(deviceMap, "device_name")
		}
		device := Device{
			ID:     id,
			Name:   name,
			Type:   getString(deviceMap, "device_type"),
			Active: id == state.activeDeviceID,
		}
		device.Volume = getInt(deviceMap, "volume")
		if device.Volume == 0 {
			device.Volume = getInt(deviceMap, "volume_percent")
		}
		device.Volume = normalizeConnectVolume(device.Volume)
		devices = append(devices, device)
	}
	return devices
}

func normalizeConnectVolume(volume int) int {
	if volume <= 100 {
		return volume
	}
	return clampVolume(int(float64(volume)*100/65535 + 0.5))
}

func mapPlaybackStatus(state connectState) PlaybackStatus {
	status := PlaybackStatus{}
	player := state.playerState
	if player == nil {
		return status
	}
	if paused, ok := player["is_paused"].(bool); ok {
		status.IsPlaying = !paused
	} else {
		status.IsPlaying = getBool(player, "is_playing")
	}
	status.ProgressMS = getInt(player, "position_as_of_timestamp")
	if status.ProgressMS == 0 {
		status.ProgressMS = getInt(player, "position_ms")
	}
	status.Shuffle = getBool(player, "shuffle")
	status.Repeat = getString(player, "repeat_mode")
	if status.Repeat == "" {
		status.Repeat = getString(player, "repeat")
	}
	if track := extractPlaybackTrack(player); track.URI != "" {
		status.Item = &track
	}
	for _, device := range mapDevices(state) {
		if device.Active {
			status.Device = device
			break
		}
	}
	return status
}

func mapQueue(state connectState) Queue {
	queue := Queue{}
	if state.playerState == nil {
		return queue
	}
	if current := extractPlaybackTrack(state.playerState); current.URI != "" {
		queue.CurrentlyPlaying = &current
	}
	if next, ok := state.playerState["next_tracks"].([]any); ok {
		for _, entry := range next {
			if item, ok := extractConnectTrack(entry); ok {
				queue.Queue = append(queue.Queue, item)
			}
		}
	}
	return queue
}

func extractPlaybackTrack(player map[string]any) Item {
	if player == nil {
		return Item{}
	}
	for _, key := range []string{"track", "item", "current_track"} {
		if raw, ok := player[key]; ok {
			if item, ok := extractConnectTrack(raw); ok {
				return item
			}
		}
	}
	for _, key := range []string{"context_uri", "context_uri_string"} {
		if uri, ok := player[key].(string); ok && strings.HasPrefix(uri, "spotify:") {
			item := Item{
				URI:  uri,
				ID:   idFromURI(uri),
				Type: typeFromURI(uri),
			}
			return item
		}
	}
	return Item{}
}

func extractConnectTrack(value any) (Item, bool) {
	item, ok := extractItem(value, "track")
	if !ok {
		return Item{}, false
	}
	entry, ok := value.(map[string]any)
	if !ok {
		return item, true
	}
	metadata, ok := getMap(entry, "metadata")
	if !ok {
		metadata, ok = getMap(entry, "track", "metadata")
	}
	if !ok {
		return item, true
	}
	if title := getString(metadata, "title"); title != "" {
		item.Name = title
	}
	if artists := connectMetadataArtists(metadata); len(artists) > 0 {
		item.Artists = artists
	}
	if album := getString(metadata, "album_title"); album != "" {
		item.Album = album
	}
	if duration, err := strconv.Atoi(getString(metadata, "duration")); err == nil && duration >= 0 {
		item.DurationMS = duration
	}
	if explicit, err := strconv.ParseBool(getString(metadata, "is_explicit")); err == nil {
		item.Explicit = explicit
		item.ExplicitKnown = true
	}
	return item, true
}

func connectMetadataArtists(metadata map[string]any) []string {
	type indexedArtist struct {
		index int
		name  string
	}
	artists := make([]indexedArtist, 0)
	for key, value := range metadata {
		name, ok := value.(string)
		if !ok || strings.TrimSpace(name) == "" {
			continue
		}
		if key == "artist_name" {
			artists = append(artists, indexedArtist{index: 0, name: name})
			continue
		}
		suffix, ok := strings.CutPrefix(key, "artist_name:")
		if !ok {
			continue
		}
		if index, err := strconv.Atoi(suffix); err == nil && index > 0 {
			artists = append(artists, indexedArtist{index: index, name: name})
		}
	}
	sort.Slice(artists, func(i, j int) bool { return artists[i].index < artists[j].index })
	names := make([]string, 0, len(artists))
	for _, artist := range artists {
		names = append(names, artist.name)
	}
	return dedupeStrings(names)
}

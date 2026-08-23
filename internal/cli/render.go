package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
)

func renderItems(w *output.Writer, items []spotify.Item) (plain []string, human []string) {
	plain = make([]string, 0, len(items))
	human = make([]string, 0, len(items))
	for _, item := range items {
		plain = append(plain, itemPlain(item))
		human = append(human, itemHuman(w, item))
	}
	return plain, human
}

func itemPlain(item spotify.Item) string {
	switch item.Type {
	case "track":
		return fmt.Sprintf("track\t%s\t%s\t%s\t%s\t%s", item.ID, item.Name, strings.Join(item.Artists, ", "), item.Album, item.URI)
	case "album":
		return fmt.Sprintf("album\t%s\t%s\t%s\t%s\t%d", item.ID, item.Name, strings.Join(item.Artists, ", "), item.ReleaseDate, item.TotalTracks)
	case "artist":
		return fmt.Sprintf("artist\t%s\t%s\t%d", item.ID, item.Name, item.Followers)
	case "playlist":
		return fmt.Sprintf("playlist\t%s\t%s\t%s\t%d", item.ID, item.Name, item.Owner, item.TotalTracks)
	case "show":
		return fmt.Sprintf("show\t%s\t%s\t%s\t%d", item.ID, item.Name, item.Publisher, item.TotalEpisodes)
	case "episode":
		return fmt.Sprintf("episode\t%s\t%s\t%d", item.ID, item.Name, item.DurationMS)
	default:
		return fmt.Sprintf("item\t%s\t%s\t%s", item.ID, item.Name, item.URI)
	}
}

func itemHuman(w *output.Writer, item spotify.Item) string {
	switch item.Type {
	case "track":
		return humanItemLine(w, item.Name, strings.Join(item.Artists, ", "), item.Album)
	case "album":
		return humanItemLine(w, item.Name, strings.Join(item.Artists, ", "), item.ReleaseDate)
	case "artist":
		followers := ""
		if item.Followers > 0 {
			followers = formatThousands(item.Followers) + " " + pluralNoun(item.Followers, "follower")
		}
		return humanItemLine(w, item.Name, "", followers)
	case "playlist":
		tracks := ""
		if item.TotalTracks > 0 {
			tracks = fmt.Sprintf("%d %s", item.TotalTracks, pluralNoun(item.TotalTracks, "track"))
		}
		return humanItemLine(w, item.Name, item.Owner, tracks)
	case "show":
		episodes := ""
		if item.TotalEpisodes > 0 {
			episodes = fmt.Sprintf("%d %s", item.TotalEpisodes, pluralNoun(item.TotalEpisodes, "episode"))
		}
		return humanItemLine(w, item.Name, item.Publisher, episodes)
	case "episode":
		duration := ""
		if item.DurationMS > 0 {
			duration = humanDuration(item.DurationMS)
		}
		return humanItemLine(w, item.Name, item.Show, duration)
	default:
		return w.Theme.Accent(item.Name)
	}
}

func pluralNoun(count int, singular string) string {
	if count == 1 {
		return singular
	}
	return singular + "s"
}

func humanItemLine(w *output.Writer, name, secondary string, details ...string) string {
	var line strings.Builder
	line.WriteString(w.Theme.Accent(name))
	if secondary != "" {
		line.WriteString(" — ")
		line.WriteString(secondary)
	}
	for _, detail := range details {
		if detail == "" {
			continue
		}
		line.WriteByte(' ')
		line.WriteString(w.Theme.Muted("· " + detail))
	}
	return line.String()
}

func formatThousands(number int) string {
	digits := strconv.Itoa(number)
	if len(digits) <= 3 {
		return digits
	}
	var formatted strings.Builder
	formatted.Grow(len(digits) + (len(digits)-1)/3)
	for index, digit := range digits {
		if index > 0 && (len(digits)-index)%3 == 0 {
			formatted.WriteByte(',')
		}
		formatted.WriteRune(digit)
	}
	return formatted.String()
}

func humanDuration(ms int) string {
	if ms <= 0 {
		return "0s"
	}
	d := time.Duration(ms) * time.Millisecond
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func playbackPlain(status spotify.PlaybackStatus) string {
	track := ""
	if status.Item != nil {
		track = status.Item.Name
	}
	return fmt.Sprintf("%t\t%d\t%s\t%s", status.IsPlaying, status.ProgressMS, status.Device.Name, track)
}

func playbackHuman(w *output.Writer, status spotify.PlaybackStatus) string {
	state := "paused"
	if status.IsPlaying {
		state = "playing"
	}
	line := w.Theme.Accent(strings.ToUpper(state))
	if status.Item != nil && status.Item.Name != "" {
		// Singles repeat the track name as the album title, so skip the duplicate.
		album := status.Item.Album
		if strings.EqualFold(album, status.Item.Name) {
			album = ""
		}
		line += " " + humanItemLine(w, status.Item.Name, strings.Join(status.Item.Artists, ", "), album)
	}
	if status.Device.Name != "" {
		line += " " + w.Theme.Muted("· "+status.Device.Name)
	}
	return line
}

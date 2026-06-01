//go:build darwin
// +build darwin

package spotify

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type AppleScriptClient struct {
	fallback API
}

type AppleScriptOptions struct {
	Fallback API
}

func NewAppleScriptClient(opts AppleScriptOptions) (API, error) {
	return &AppleScriptClient{
		fallback: opts.Fallback,
	}, nil
}

func (c *AppleScriptClient) runScript(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", fmt.Errorf("applescript error: %w", err)
		}
		return "", fmt.Errorf("applescript error: %w (%s)", err, msg)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *AppleScriptClient) Play(ctx context.Context, uri string) error {
	var script string
	if uri == "" {
		script = `tell application "Spotify" to play`
	} else {
		script = fmt.Sprintf(`tell application "Spotify" to play track "%s"`, uri)
	}
	_, err := c.runScript(ctx, script)
	return err
}

func (c *AppleScriptClient) Pause(ctx context.Context) error {
	_, err := c.runScript(ctx, `tell application "Spotify" to pause`)
	return err
}

func (c *AppleScriptClient) Next(ctx context.Context) error {
	_, err := c.runScript(ctx, `tell application "Spotify" to next track`)
	return err
}

func (c *AppleScriptClient) Previous(ctx context.Context) error {
	_, err := c.runScript(ctx, `tell application "Spotify" to previous track`)
	return err
}

func (c *AppleScriptClient) Seek(ctx context.Context, positionMS int) error {
	positionSec := positionMS / 1000
	script := fmt.Sprintf(`tell application "Spotify" to set player position to %d`, positionSec)
	_, err := c.runScript(ctx, script)
	return err
}

func (c *AppleScriptClient) Volume(ctx context.Context, volume int) error {
	script := fmt.Sprintf(`tell application "Spotify" to set sound volume to %d`, volume)
	_, err := c.runScript(ctx, script)
	return err
}

func (c *AppleScriptClient) Shuffle(ctx context.Context, enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}
	script := fmt.Sprintf(`tell application "Spotify" to set shuffling to %s`, val)
	_, err := c.runScript(ctx, script)
	return err
}

func (c *AppleScriptClient) Repeat(ctx context.Context, mode string) error {
	val := "false"
	if mode == "track" || mode == "context" {
		val = "true"
	}
	script := fmt.Sprintf(`tell application "Spotify" to set repeating to %s`, val)
	_, err := c.runScript(ctx, script)
	return err
}

func (c *AppleScriptClient) Playback(ctx context.Context) (PlaybackStatus, error) {
	script := `tell application "Spotify"
	set trackName to name of current track
	set trackArtist to artist of current track
	set trackAlbum to album of current track
	set trackID to id of current track
	set trackDuration to duration of current track
	set playerPos to player position
	set playerState to player state as string
	set vol to sound volume
	set isShuffling to shuffling
	set isRepeating to repeating
	return trackName & "|||" & trackArtist & "|||" & trackAlbum & "|||" & trackID & "|||" & trackDuration & "|||" & playerPos & "|||" & playerState & "|||" & vol & "|||" & isShuffling & "|||" & isRepeating
end tell`
	out, err := c.runScript(ctx, script)
	if err != nil {
		return PlaybackStatus{}, err
	}
	parts := strings.Split(out, "|||")
	if len(parts) < 10 {
		return PlaybackStatus{}, fmt.Errorf("unexpected applescript output: %s", out)
	}
	durationMS, _ := strconv.Atoi(parts[4])
	positionSec, _ := strconv.ParseFloat(parts[5], 64)
	volume, _ := strconv.Atoi(parts[7])
	isPlaying := parts[6] == "playing"
	shuffle := parts[8] == "true"
	repeat := "off"
	if parts[9] == "true" {
		repeat = "context"
	}
	item := &Item{
		URI:        parts[3],
		Name:       parts[0],
		Artists:    []string{parts[1]},
		Album:      parts[2],
		DurationMS: durationMS,
	}
	return PlaybackStatus{
		IsPlaying:  isPlaying,
		ProgressMS: int(positionSec * 1000),
		Item:       item,
		Device: Device{
			ID:     "local",
			Name:   "Local Spotify",
			Type:   "COMPUTER",
			Volume: volume,
			Active: true,
		},
		Shuffle: shuffle,
		Repeat:  repeat,
	}, nil
}

func (c *AppleScriptClient) Devices(ctx context.Context) ([]Device, error) {
	return []Device{
		{
			ID:     "local",
			Name:   "Local Spotify",
			Type:   "COMPUTER",
			Active: true,
		},
	}, nil
}

func (c *AppleScriptClient) Transfer(ctx context.Context, deviceID string) error {
	return ErrUnsupported
}

func (c *AppleScriptClient) QueueAdd(ctx context.Context, uri string) error {
	if c.fallback != nil {
		return c.fallback.QueueAdd(ctx, uri)
	}
	return ErrUnsupported
}

func (c *AppleScriptClient) Queue(ctx context.Context) (Queue, error) {
	if c.fallback != nil {
		return c.fallback.Queue(ctx)
	}
	return Queue{}, ErrUnsupported
}

func (c *AppleScriptClient) Search(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	if c.fallback != nil {
		return c.fallback.Search(ctx, kind, query, limit, offset)
	}
	return SearchResult{}, ErrUnsupported
}

func (c *AppleScriptClient) GetTrack(ctx context.Context, id string) (Item, error) {
	if c.fallback != nil {
		return c.fallback.GetTrack(ctx, id)
	}
	return Item{}, ErrUnsupported
}

func (c *AppleScriptClient) GetAlbum(ctx context.Context, id string) (Item, error) {
	if c.fallback != nil {
		return c.fallback.GetAlbum(ctx, id)
	}
	return Item{}, ErrUnsupported
}

func (c *AppleScriptClient) GetArtist(ctx context.Context, id string) (Item, error) {
	if c.fallback != nil {
		return c.fallback.GetArtist(ctx, id)
	}
	return Item{}, ErrUnsupported
}

func (c *AppleScriptClient) GetPlaylist(ctx context.Context, id string) (Item, error) {
	if c.fallback != nil {
		return c.fallback.GetPlaylist(ctx, id)
	}
	return Item{}, ErrUnsupported
}

func (c *AppleScriptClient) GetShow(ctx context.Context, id string) (Item, error) {
	if c.fallback != nil {
		return c.fallback.GetShow(ctx, id)
	}
	return Item{}, ErrUnsupported
}

func (c *AppleScriptClient) GetEpisode(ctx context.Context, id string) (Item, error) {
	if c.fallback != nil {
		return c.fallback.GetEpisode(ctx, id)
	}
	return Item{}, ErrUnsupported
}

func (c *AppleScriptClient) LibraryTracks(ctx context.Context, limit, offset int) ([]Item, int, error) {
	if c.fallback != nil {
		return c.fallback.LibraryTracks(ctx, limit, offset)
	}
	return nil, 0, ErrUnsupported
}

func (c *AppleScriptClient) LibraryAlbums(ctx context.Context, limit, offset int) ([]Item, int, error) {
	if c.fallback != nil {
		return c.fallback.LibraryAlbums(ctx, limit, offset)
	}
	return nil, 0, ErrUnsupported
}

func (c *AppleScriptClient) LibraryModify(ctx context.Context, path string, ids []string, method string) error {
	if c.fallback != nil {
		return c.fallback.LibraryModify(ctx, path, ids, method)
	}
	return ErrUnsupported
}

func (c *AppleScriptClient) FollowArtists(ctx context.Context, ids []string, method string) error {
	if c.fallback != nil {
		return c.fallback.FollowArtists(ctx, ids, method)
	}
	return ErrUnsupported
}

func (c *AppleScriptClient) FollowedArtists(ctx context.Context, limit int, after string) ([]Item, int, string, error) {
	if c.fallback != nil {
		return c.fallback.FollowedArtists(ctx, limit, after)
	}
	return nil, 0, "", ErrUnsupported
}

func (c *AppleScriptClient) Playlists(ctx context.Context, limit, offset int) ([]Item, int, error) {
	if c.fallback != nil {
		return c.fallback.Playlists(ctx, limit, offset)
	}
	return nil, 0, ErrUnsupported
}

func (c *AppleScriptClient) PlaylistTracks(ctx context.Context, id string, limit, offset int) ([]Item, int, error) {
	if c.fallback != nil {
		return c.fallback.PlaylistTracks(ctx, id, limit, offset)
	}
	return nil, 0, ErrUnsupported
}

func (c *AppleScriptClient) CreatePlaylist(ctx context.Context, name string, public, collaborative bool) (Item, error) {
	if c.fallback != nil {
		return c.fallback.CreatePlaylist(ctx, name, public, collaborative)
	}
	return Item{}, ErrUnsupported
}

func (c *AppleScriptClient) AddTracks(ctx context.Context, playlistID string, uris []string) error {
	if c.fallback != nil {
		return c.fallback.AddTracks(ctx, playlistID, uris)
	}
	return ErrUnsupported
}

func (c *AppleScriptClient) RemoveTracks(ctx context.Context, playlistID string, uris []string) error {
	if c.fallback != nil {
		return c.fallback.RemoveTracks(ctx, playlistID, uris)
	}
	return ErrUnsupported
}

func (c *AppleScriptClient) GetUsersTopTracks(ctx context.Context, timeRange string, limit, offset int) (TopTracksResult, error) {
	if c.fallback != nil {
		return c.fallback.GetUsersTopTracks(ctx, timeRange, limit, offset)
	}
	return TopTracksResult{}, ErrUnsupported
}

func (c *AppleScriptClient) GetRecentlyPlayed(ctx context.Context, limit int, after, before int64) (RecentlyPlayedResult, error) {
	if c.fallback != nil {
		return c.fallback.GetRecentlyPlayed(ctx, limit, after, before)
	}
	return RecentlyPlayedResult{}, ErrUnsupported
}

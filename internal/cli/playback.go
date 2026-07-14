package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/steipete/spogo/internal/app"
	"github.com/steipete/spogo/internal/spotify"
)

type PlayCmd struct {
	Items   []string `arg:"" optional:"" help:"Spotify ID/URL/URI. Pass two or more track URIs to start an ordered track list."`
	Type    string   `help:"Type for raw IDs (track|album|playlist|show|episode)."`
	Shuffle bool     `help:"Enable shuffle before playing."`
}

type PauseCmd struct{}

type NextCmd struct{}

type PrevCmd struct{}

type SeekCmd struct {
	Position string `arg:"" required:"" help:"Position ms or mm:ss."`
}

type VolumeCmd struct {
	Level int `arg:"" required:"" help:"Volume 0-100."`
}

type ShuffleCmd struct {
	State string `arg:"" required:"" help:"on|off."`
}

type RepeatCmd struct {
	Mode string `arg:"" required:"" help:"off|track|context."`
}

type StatusCmd struct{}

type artistTopTracks interface {
	ArtistTopTracks(ctx context.Context, id string, limit int) ([]spotify.Item, error)
}

func (cmd *PlayCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	// Two or more arguments start an ordered list of tracks in a single request
	// (not a queue append). One argument keeps the existing single-item behavior
	// (track/context/artist top-track); no argument resumes.
	if len(cmd.Items) > 1 {
		return cmd.runTracks(ctx, cmdCtx, client)
	}
	uri := ""
	if len(cmd.Items) == 1 {
		uri, err = resolvePlayURI(cmdCtx, client, cmd.Items[0], cmd.Type)
		if err != nil {
			return err
		}
	}
	if cmd.Shuffle {
		if err := client.Shuffle(cmdCtx, true); err != nil {
			return err
		}
	}
	if err := client.Play(cmdCtx, uri); err != nil {
		return err
	}
	return emitOK(ctx, nil, "Playback started")
}

// runTracks starts an ordered list of track URIs. Every argument must resolve to
// a track; contexts (album/playlist/show/episode) and artists are rejected so
// the ordered list stays unambiguous.
func (cmd *PlayCmd) runTracks(ctx *app.Context, cmdCtx context.Context, client spotify.API) error {
	uris := make([]string, 0, len(cmd.Items))
	for _, item := range cmd.Items {
		uri, err := resolveTrackURI(item, cmd.Type)
		if err != nil {
			return err
		}
		uris = append(uris, uri)
	}
	if cmd.Shuffle {
		if err := client.Shuffle(cmdCtx, true); err != nil {
			return err
		}
	}
	if err := client.PlayTracks(cmdCtx, uris); err != nil {
		return err
	}
	return emitOK(ctx, map[string]any{"status": "ok", "count": len(uris)}, fmt.Sprintf("Playing %d tracks", len(uris)))
}

// resolvePlayURI resolves a single play target, preserving artist top-track and
// raw-id-with-type behavior.
func resolvePlayURI(ctx context.Context, client spotify.API, item, typeFlag string) (string, error) {
	res, err := spotify.ParseResource(item)
	if err != nil {
		return "", err
	}
	if res.URI == "" {
		if typeFlag == "" {
			return "", errors.New("type required for raw id")
		}
		res.Type = typeFlag
		res.URI = "spotify:" + typeFlag + ":" + res.ID
	}
	if res.Type != "artist" {
		return res.URI, nil
	}
	return resolveArtistURI(ctx, client, res)
}

func resolveArtistURI(ctx context.Context, client spotify.API, res spotify.Resource) (string, error) {
	topTracks, ok := client.(artistTopTracks)
	if !ok {
		return "", errors.New("artist playback not supported by engine")
	}
	tracks, err := topTracks.ArtistTopTracks(ctx, res.ID, 10)
	if err == nil && len(tracks) > 0 {
		return tracks[0].URI, nil
	}
	artist, aerr := client.GetArtist(ctx, res.ID)
	if aerr != nil || artist.Name == "" {
		if err != nil {
			return "", err
		}
		return "", errors.New("no artist tracks found")
	}
	query := fmt.Sprintf("artist:%q", artist.Name)
	search, serr := client.Search(ctx, "track", query, 1, 0)
	if serr != nil {
		if err != nil {
			return "", err
		}
		return "", serr
	}
	if len(search.Items) == 0 {
		if err != nil {
			return "", err
		}
		return "", errors.New("no artist tracks found")
	}
	return search.Items[0].URI, nil
}

// resolveTrackURI resolves one argument to a track URI for ordered multi-track
// play. Raw IDs require --type track; any non-track resource is rejected.
func resolveTrackURI(item, typeFlag string) (string, error) {
	res, err := spotify.ParseResource(item)
	if err != nil {
		return "", err
	}
	if res.URI == "" {
		if typeFlag == "" {
			return "", errors.New("type required for raw id")
		}
		if typeFlag != "track" {
			return "", fmt.Errorf("multi-track play only supports tracks, got type %q", typeFlag)
		}
		return "spotify:track:" + res.ID, nil
	}
	if res.Type != "track" {
		return "", fmt.Errorf("multi-track play only supports tracks, got %s %q", res.Type, item)
	}
	return res.URI, nil
}

func (cmd *PauseCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	if err := client.Pause(cmdCtx); err != nil {
		return err
	}
	return emitOK(ctx, nil, "Playback paused")
}

func (cmd *NextCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	if err := client.Next(cmdCtx); err != nil {
		return err
	}
	return emitOK(ctx, nil, "Skipped to next")
}

func (cmd *PrevCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	if err := client.Previous(cmdCtx); err != nil {
		return err
	}
	return emitOK(ctx, nil, "Skipped to previous")
}

func (cmd *SeekCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	position, err := parsePosition(cmd.Position)
	if err != nil {
		return err
	}
	if err := client.Seek(cmdCtx, position); err != nil {
		return err
	}
	return emitOK(ctx, map[string]any{"status": "ok", "position_ms": position}, fmt.Sprintf("Seeked to %s", humanDuration(position)))
}

func (cmd *VolumeCmd) Run(ctx *app.Context) error {
	if cmd.Level < 0 || cmd.Level > 100 {
		return fmt.Errorf("volume must be 0-100")
	}
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	if err := client.Volume(cmdCtx, cmd.Level); err != nil {
		return err
	}
	return emitOK(ctx, map[string]any{"status": "ok", "volume": cmd.Level}, fmt.Sprintf("Volume %d", cmd.Level))
}

func (cmd *ShuffleCmd) Run(ctx *app.Context) error {
	state, err := parseToggle(cmd.State)
	if err != nil {
		return err
	}
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	if err := client.Shuffle(cmdCtx, state); err != nil {
		return err
	}
	return emitOK(ctx, map[string]any{"status": "ok", "shuffle": state}, fmt.Sprintf("Shuffle %s", onOff(state)))
}

func (cmd *RepeatCmd) Run(ctx *app.Context) error {
	mode := strings.ToLower(strings.TrimSpace(cmd.Mode))
	if mode != "off" && mode != "track" && mode != "context" {
		return fmt.Errorf("repeat must be off|track|context")
	}
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	if err := client.Repeat(cmdCtx, mode); err != nil {
		return err
	}
	return emitOK(ctx, map[string]any{"status": "ok", "repeat": mode}, fmt.Sprintf("Repeat %s", mode))
}

func (cmd *StatusCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	status, err := client.Playback(cmdCtx)
	if err != nil {
		return err
	}
	plain := []string{playbackPlain(status)}
	human := []string{playbackHuman(ctx.Output, status)}
	return ctx.Output.Emit(status, plain, human)
}

func parsePosition(input string) (int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, fmt.Errorf("position required")
	}
	if strings.Contains(input, ":") {
		parts := strings.Split(input, ":")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid position format")
		}
		min, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, err
		}
		sec, err := strconv.Atoi(parts[1])
		if err != nil {
			return 0, err
		}
		d := time.Duration(min)*time.Minute + time.Duration(sec)*time.Second
		return int(d / time.Millisecond), nil
	}
	ms, err := strconv.Atoi(input)
	if err != nil {
		return 0, err
	}
	return ms, nil
}

func parseToggle(input string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "on", "true", "1", "yes":
		return true, nil
	case "off", "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("expected on|off")
	}
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

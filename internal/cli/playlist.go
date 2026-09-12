package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/steipete/spogo/internal/app"
	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
)

type PlaylistCreateCmd struct {
	Name   string `arg:"" required:"" help:"Playlist name."`
	Public bool   `help:"Create public playlist."`
	Collab bool   `help:"Create collaborative playlist."`
}

type PlaylistAddCmd struct {
	Playlist string   `arg:"" required:"" help:"Playlist ID/URL/URI."`
	Tracks   []string `arg:"" required:"" help:"Track IDs/URLs/URIs."`
}

type PlaylistRemoveCmd struct {
	Playlist string   `arg:"" required:"" help:"Playlist ID/URL/URI."`
	Tracks   []string `arg:"" required:"" help:"Track IDs/URLs/URIs."`
}

type PlaylistTracksCmd struct {
	Playlist string `arg:"" required:"" help:"Playlist ID/URL/URI."`
	Limit    int    `help:"Limit results." default:"50"`
	Offset   int    `help:"Offset results." default:"0"`
}

type PlaylistFollowCmd struct {
	Playlist string `arg:"" required:"" help:"Playlist ID/URL/URI."`
}

type PlaylistUnfollowCmd struct {
	Playlist string `arg:"" required:"" help:"Playlist ID/URL/URI."`
}

type PlaylistFollowingCmd struct {
	Playlist string `arg:"" required:"" help:"Playlist ID/URL/URI."`
}

func (cmd *PlaylistCreateCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	item, err := client.CreatePlaylist(cmdCtx, cmd.Name, cmd.Public, cmd.Collab)
	if err != nil {
		return err
	}
	return ctx.Output.Emit(item, []string{itemPlain(item)}, []string{fmt.Sprintf("Created %s", itemHuman(ctx.Output, item))})
}

func (cmd *PlaylistAddCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	playlist, err := spotify.ParseTypedID(cmd.Playlist, "playlist")
	if err != nil {
		return err
	}
	uris, err := trackURIs(cmd.Tracks)
	if err != nil {
		return err
	}
	if err := client.AddTracks(cmdCtx, playlist.ID, uris); err != nil {
		return err
	}
	return emitCountStatus(ctx, len(uris), "Added")
}

func (cmd *PlaylistRemoveCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	playlist, err := spotify.ParseTypedID(cmd.Playlist, "playlist")
	if err != nil {
		return err
	}
	uris, err := trackURIs(cmd.Tracks)
	if err != nil {
		return err
	}
	if err := client.RemoveTracks(cmdCtx, playlist.ID, uris); err != nil {
		return err
	}
	return emitCountStatus(ctx, len(uris), "Removed")
}

func (cmd *PlaylistTracksCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	playlist, err := spotify.ParseTypedID(cmd.Playlist, "playlist")
	if err != nil {
		return err
	}
	limit := clampLimit(cmd.Limit)
	items, total, err := client.PlaylistTracks(cmdCtx, playlist.ID, limit, cmd.Offset)
	if err != nil {
		return err
	}
	plain, human := renderItems(ctx.Output, items)
	if ctx.Output.Format == output.FormatHuman {
		human = append([]string{fmt.Sprintf("Tracks: %d", total)}, human...)
	}
	payload := map[string]any{"total": total, "items": items}
	return ctx.Output.Emit(payload, plain, human)
}

func (cmd *PlaylistFollowCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	playlist, err := spotify.ParseTypedID(cmd.Playlist, "playlist")
	if err != nil {
		return err
	}
	if err := client.FollowPlaylist(cmdCtx, playlist.ID); err != nil {
		return err
	}
	name := playlist.ID
	if ctx.Output.Format == output.FormatHuman {
		name = playlistDisplayName(cmdCtx, client, playlist.ID)
	}
	return emitOK(ctx, map[string]any{"status": "ok", "id": playlist.ID}, fmt.Sprintf("Followed %s", name))
}

func (cmd *PlaylistUnfollowCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	playlist, err := spotify.ParseTypedID(cmd.Playlist, "playlist")
	if err != nil {
		return err
	}
	if err := client.UnfollowPlaylist(cmdCtx, playlist.ID); err != nil {
		return err
	}
	name := playlist.ID
	if ctx.Output.Format == output.FormatHuman {
		name = playlistDisplayName(cmdCtx, client, playlist.ID)
	}
	return emitOK(ctx, map[string]any{"status": "ok", "id": playlist.ID}, fmt.Sprintf("Unfollowed %s", name))
}

func (cmd *PlaylistFollowingCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	playlist, err := spotify.ParseTypedID(cmd.Playlist, "playlist")
	if err != nil {
		return err
	}
	following, err := client.IsFollowingPlaylist(cmdCtx, playlist.ID)
	if err != nil {
		return err
	}
	payload := map[string]any{"following": following}
	return ctx.Output.Emit(payload, []string{fmt.Sprint(following)}, []string{fmt.Sprintf("Following: %v", following)})
}

func playlistDisplayName(ctx context.Context, client spotify.API, id string) string {
	item, err := client.GetPlaylist(ctx, id)
	if err != nil || item.Name == "" {
		return id
	}
	return item.Name
}

func trackURIs(inputs []string) ([]string, error) {
	uris := make([]string, 0, len(inputs))
	for _, input := range inputs {
		res, err := spotify.ParseTypedID(strings.TrimSpace(input), "track")
		if err != nil {
			return nil, err
		}
		if res.URI == "" {
			return nil, fmt.Errorf("invalid track input")
		}
		uris = append(uris, res.URI)
	}
	return uris, nil
}

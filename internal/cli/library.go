package cli

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/steipete/spogo/internal/app"
	"github.com/steipete/spogo/internal/spotify"
)

type LibraryCmd struct {
	Tracks    LibraryTracksCmd    `kong:"cmd,help='Track library.'"`
	Albums    LibraryAlbumsCmd    `kong:"cmd,help='Album library.'"`
	Artists   LibraryArtistsCmd   `kong:"cmd,help='Artist library.'"`
	Playlists LibraryPlaylistsCmd `kong:"cmd,help='Playlist library.'"`
}

type LibraryTracksCmd struct {
	List   LibraryTracksListCmd   `kong:"cmd,help='List saved tracks.'"`
	Add    LibraryTracksAddCmd    `kong:"cmd,help='Save tracks.'"`
	Remove LibraryTracksRemoveCmd `kong:"cmd,help='Remove saved tracks.'"`
}

type LibraryAlbumsCmd struct {
	List   LibraryAlbumsListCmd   `kong:"cmd,help='List saved albums.'"`
	Add    LibraryAlbumsAddCmd    `kong:"cmd,help='Save albums.'"`
	Remove LibraryAlbumsRemoveCmd `kong:"cmd,help='Remove saved albums.'"`
}

type LibraryArtistsCmd struct {
	List     LibraryArtistsListCmd     `kong:"cmd,help='List followed artists.'"`
	Follow   LibraryArtistsFollowCmd   `kong:"cmd,help='Follow artists.'"`
	Unfollow LibraryArtistsUnfollowCmd `kong:"cmd,help='Unfollow artists.'"`
}

type LibraryPlaylistsCmd struct {
	List LibraryPlaylistsListCmd `kong:"cmd,help='List playlists.'"`
}

type LibraryTracksListCmd struct {
	Limit  int `help:"Limit results." default:"50"`
	Offset int `help:"Offset results." default:"0"`
}

type LibraryTracksAddCmd struct {
	IDs []string `arg:"" required:"" help:"Track IDs/URLs/URIs."`
}

type LibraryTracksRemoveCmd struct {
	IDs []string `arg:"" required:"" help:"Track IDs/URLs/URIs."`
}

type LibraryAlbumsListCmd struct {
	Limit  int `help:"Limit results." default:"50"`
	Offset int `help:"Offset results." default:"0"`
}

type LibraryAlbumsAddCmd struct {
	IDs []string `arg:"" required:"" help:"Album IDs/URLs/URIs."`
}

type LibraryAlbumsRemoveCmd struct {
	IDs []string `arg:"" required:"" help:"Album IDs/URLs/URIs."`
}

type LibraryArtistsListCmd struct {
	Limit  int    `help:"Limit results." default:"50"`
	After  string `help:"Artist ID to start after (pagination)."`
	Offset int    `help:"Offset results (not supported by Spotify)."`
}

type LibraryArtistsFollowCmd struct {
	IDs []string `arg:"" required:"" help:"Artist IDs/URLs/URIs."`
}

type LibraryArtistsUnfollowCmd struct {
	IDs []string `arg:"" required:"" help:"Artist IDs/URLs/URIs."`
}

type LibraryPlaylistsListCmd struct {
	Limit  int `help:"Limit results." default:"50"`
	Offset int `help:"Offset results." default:"0"`
}

func (cmd *LibraryTracksListCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	limit := clampLimit(cmd.Limit)
	items, total, err := client.LibraryTracks(cmdCtx, limit, cmd.Offset)
	if err != nil {
		return err
	}
	return emitItems(ctx, items, total, nil)
}

func (cmd *LibraryTracksAddCmd) Run(ctx *app.Context) error {
	return runLibraryModify(ctx, cmd.IDs, "track", http.MethodPut)
}

func (cmd *LibraryTracksRemoveCmd) Run(ctx *app.Context) error {
	return runLibraryModify(ctx, cmd.IDs, "track", http.MethodDelete)
}

func (cmd *LibraryAlbumsListCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	limit := clampLimit(cmd.Limit)
	items, total, err := client.LibraryAlbums(cmdCtx, limit, cmd.Offset)
	if err != nil {
		return err
	}
	return emitItems(ctx, items, total, nil)
}

func (cmd *LibraryAlbumsAddCmd) Run(ctx *app.Context) error {
	return runLibraryModify(ctx, cmd.IDs, "album", http.MethodPut)
}

func (cmd *LibraryAlbumsRemoveCmd) Run(ctx *app.Context) error {
	return runLibraryModify(ctx, cmd.IDs, "album", http.MethodDelete)
}

func (cmd *LibraryArtistsListCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	if cmd.Offset != 0 {
		return fmt.Errorf("offset not supported; use --after with an artist id")
	}
	limit := clampLimit(cmd.Limit)
	items, total, next, err := client.FollowedArtists(cmdCtx, limit, cmd.After)
	if err != nil {
		return err
	}
	plain, human := renderItems(ctx.Output, items)
	payload := map[string]any{"total": total, "items": items, "next_after": next}
	return ctx.Output.Emit(payload, plain, human)
}

func (cmd *LibraryArtistsFollowCmd) Run(ctx *app.Context) error {
	return runLibraryModify(ctx, cmd.IDs, "artist", http.MethodPut)
}

func (cmd *LibraryArtistsUnfollowCmd) Run(ctx *app.Context) error {
	return runLibraryModify(ctx, cmd.IDs, "artist", http.MethodDelete)
}

func (cmd *LibraryPlaylistsListCmd) Run(ctx *app.Context) error {
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	limit := clampLimit(cmd.Limit)
	items, total, err := client.Playlists(cmdCtx, limit, cmd.Offset)
	if err != nil {
		return err
	}
	return emitItems(ctx, items, total, nil)
}

func runLibraryModify(ctx *app.Context, inputs []string, kind, method string) error {
	ids, err := parseIDs(inputs, kind)
	if err != nil {
		return err
	}
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	if kind == "artist" {
		err = client.FollowArtists(cmdCtx, ids, method)
	} else {
		err = client.LibraryModify(cmdCtx, "/me/"+kind+"s", ids, method)
	}
	if err != nil {
		return err
	}
	return emitCountStatus(ctx, len(ids), "Updated")
}

func parseIDs(inputs []string, kind string) ([]string, error) {
	ids := make([]string, 0, len(inputs))
	for _, input := range inputs {
		res, err := spotify.ParseTypedID(strings.TrimSpace(input), kind)
		if err != nil {
			return nil, err
		}
		ids = append(ids, res.ID)
	}
	return ids, nil
}

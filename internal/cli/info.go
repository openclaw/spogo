package cli

import (
	"context"

	"github.com/steipete/spogo/internal/app"
	"github.com/steipete/spogo/internal/spotify"
)

type TrackCmd struct {
	Info InfoTrackCmd `kong:"cmd,help='Track info.'"`
}

type AlbumCmd struct {
	Info InfoAlbumCmd `kong:"cmd,help='Album info.'"`
}

type ArtistCmd struct {
	Info InfoArtistCmd `kong:"cmd,help='Artist info.'"`
}

type PlaylistCmd struct {
	Info      InfoPlaylistCmd      `kong:"cmd,help='Playlist info.'"`
	Create    PlaylistCreateCmd    `kong:"cmd,help='Create playlist.'"`
	Add       PlaylistAddCmd       `kong:"cmd,help='Add tracks to playlist.'"`
	Remove    PlaylistRemoveCmd    `kong:"cmd,help='Remove tracks from playlist.'"`
	Tracks    PlaylistTracksCmd    `kong:"cmd,help='List playlist tracks.'"`
	Follow    PlaylistFollowCmd    `kong:"cmd,help='Follow playlist.'"`
	Unfollow  PlaylistUnfollowCmd  `kong:"cmd,help='Unfollow playlist.'"`
	Following PlaylistFollowingCmd `kong:"cmd,help='Check whether you follow a playlist.'"`
}

type ShowCmd struct {
	Info InfoShowCmd `kong:"cmd,help='Show info.'"`
}

type EpisodeCmd struct {
	Info InfoEpisodeCmd `kong:"cmd,help='Episode info.'"`
}

type InfoArgs struct {
	ID string `arg:"" required:"" help:"Spotify ID, URI, or URL."`
}

type InfoTrackCmd struct{ InfoArgs }

type InfoAlbumCmd struct{ InfoArgs }

type InfoArtistCmd struct{ InfoArgs }

type InfoPlaylistCmd struct{ InfoArgs }

type InfoShowCmd struct{ InfoArgs }

type InfoEpisodeCmd struct{ InfoArgs }

func (cmd *InfoTrackCmd) Run(ctx *app.Context) error {
	return runInfoLookup(ctx, cmd.ID, "track", func(cmdCtx context.Context, client spotify.API, id string) (spotify.Item, error) {
		return client.GetTrack(cmdCtx, id)
	})
}

func (cmd *InfoAlbumCmd) Run(ctx *app.Context) error {
	return runInfoLookup(ctx, cmd.ID, "album", func(cmdCtx context.Context, client spotify.API, id string) (spotify.Item, error) {
		return client.GetAlbum(cmdCtx, id)
	})
}

func (cmd *InfoArtistCmd) Run(ctx *app.Context) error {
	return runInfoLookup(ctx, cmd.ID, "artist", func(cmdCtx context.Context, client spotify.API, id string) (spotify.Item, error) {
		return client.GetArtist(cmdCtx, id)
	})
}

func (cmd *InfoPlaylistCmd) Run(ctx *app.Context) error {
	return runInfoLookup(ctx, cmd.ID, "playlist", func(cmdCtx context.Context, client spotify.API, id string) (spotify.Item, error) {
		return client.GetPlaylist(cmdCtx, id)
	})
}

func (cmd *InfoShowCmd) Run(ctx *app.Context) error {
	return runInfoLookup(ctx, cmd.ID, "show", func(cmdCtx context.Context, client spotify.API, id string) (spotify.Item, error) {
		return client.GetShow(cmdCtx, id)
	})
}

func (cmd *InfoEpisodeCmd) Run(ctx *app.Context) error {
	return runInfoLookup(ctx, cmd.ID, "episode", func(cmdCtx context.Context, client spotify.API, id string) (spotify.Item, error) {
		return client.GetEpisode(cmdCtx, id)
	})
}

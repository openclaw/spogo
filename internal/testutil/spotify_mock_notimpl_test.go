package testutil

import (
	"context"
	"testing"
)

func TestSpotifyMockAllNotImplemented(t *testing.T) {
	m := &SpotifyMock{}
	_, _ = m.Search(context.Background(), "track", "q", 1, 0)
	_, _ = m.GetTrack(context.Background(), "1")
	_, _ = m.GetAlbum(context.Background(), "1")
	_, _ = m.GetArtist(context.Background(), "1")
	_, _ = m.GetPlaylist(context.Background(), "1")
	_, _ = m.GetShow(context.Background(), "1")
	_, _ = m.GetEpisode(context.Background(), "1")
	_, _ = m.ArtistTopTracks(context.Background(), "1", 10)
	_, _ = m.Playback(context.Background())
	_ = m.Play(context.Background(), "uri")
	_ = m.Pause(context.Background())
	_ = m.Next(context.Background())
	_ = m.Previous(context.Background())
	_ = m.Seek(context.Background(), 1)
	_ = m.Volume(context.Background(), 1)
	_ = m.Shuffle(context.Background(), true)
	_ = m.Repeat(context.Background(), "off")
	_, _ = m.Devices(context.Background())
	_ = m.Transfer(context.Background(), "id")
	_ = m.QueueAdd(context.Background(), "uri")
	_, _ = m.Queue(context.Background())
	_, _, _ = m.LibraryTracks(context.Background(), 1, 0)
	_, _, _ = m.LibraryAlbums(context.Background(), 1, 0)
	_ = m.LibraryModify(context.Background(), "/me/tracks", []string{"1"}, "PUT")
	_ = m.FollowArtists(context.Background(), []string{"1"}, "PUT")
	_, _, _, _ = m.FollowedArtists(context.Background(), 1, "")
	_, _, _ = m.Playlists(context.Background(), 1, 0)
	_, _, _ = m.PlaylistTracks(context.Background(), "1", 1, 0)
	_, _ = m.CreatePlaylist(context.Background(), "name", true, false)
	_ = m.AddTracks(context.Background(), "p", []string{"u"})
	_ = m.RemoveTracks(context.Background(), "p", []string{"u"})
	_, _ = m.GetUsersTopTracks(context.Background(), "long_term", 20, 0)
	_, _ = m.GetRecentlyPlayed(context.Background(), 20, 0, 0)
}

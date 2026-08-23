package cli

import (
	"strings"
	"testing"

	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestRenderItemsHuman(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatHuman)
	items := []spotify.Item{{ID: "t1", Name: "Song", Type: "track", Artists: []string{"Artist"}, Album: "Album"}}
	plain, human := renderItems(ctx.Output, items)
	if len(plain) == 0 || len(human) == 0 {
		t.Fatalf("expected lines")
	}
	_ = ctx.Output.Emit(items, plain, human)
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlaybackFormatting(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatHuman)
	status := spotify.PlaybackStatus{IsPlaying: true, ProgressMS: 120000, Device: spotify.Device{Name: "Desk"}, Item: &spotify.Item{Name: "Song", Artists: []string{"Artist"}}}
	if playbackPlain(status) == "" {
		t.Fatalf("expected plain")
	}
	if playbackHuman(ctx.Output, status) == "" {
		t.Fatalf("expected human")
	}
}

func TestPlaylistRenderOmitsZeroTracks(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatHuman)
	line := itemHuman(ctx.Output, spotify.Item{Type: "playlist", Name: "List", Owner: "Peter"})
	if strings.Contains(line, "0 tracks") {
		t.Fatalf("unexpected track count: %s", line)
	}
}

func TestItemHumanOmitsUnknownMetadata(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatHuman)
	tests := []struct {
		name string
		item spotify.Item
		want string
	}{
		{"track", spotify.Item{Type: "track", Name: "Song"}, "Song"},
		{"album", spotify.Item{Type: "album", Name: "Parachutes", Artists: []string{"Coldplay"}}, "Parachutes — Coldplay"},
		{"artist", spotify.Item{Type: "artist", Name: "Weezer"}, "Weezer"},
		{"playlist", spotify.Item{Type: "playlist", Name: "Discover Weekly"}, "Discover Weekly"},
		{"show", spotify.Item{Type: "show", Name: "Darknet Diaries"}, "Darknet Diaries"},
		{"episode", spotify.Item{Type: "episode", Name: "178: Ubiquiti"}, "178: Ubiquiti"},
		{"unknown", spotify.Item{Type: "other", Name: "Other"}, "Other"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := itemHuman(ctx.Output, test.item); got != test.want {
				t.Fatalf("line=%q want=%q", got, test.want)
			}
		})
	}
}

func TestItemHumanFormatsCompleteMetadata(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatHuman)
	tests := []struct {
		item spotify.Item
		want string
	}{
		{spotify.Item{Type: "artist", Name: "Weezer", Followers: 5567807}, "Weezer · 5,567,807 followers"},
		{spotify.Item{Type: "playlist", Name: "TRANCE", Owner: "YOU LOVE DANCE", TotalTracks: 197}, "TRANCE — YOU LOVE DANCE · 197 tracks"},
		{spotify.Item{Type: "show", Name: "Darknet Diaries", Publisher: "Jack Rhysider", TotalEpisodes: 227}, "Darknet Diaries — Jack Rhysider · 227 episodes"},
		{spotify.Item{Type: "show", Name: "One-Off", TotalEpisodes: 1}, "One-Off · 1 episode"},
		{spotify.Item{Type: "playlist", Name: "Single", TotalTracks: 1}, "Single · 1 track"},
		{spotify.Item{Type: "artist", Name: "Newcomer", Followers: 1}, "Newcomer · 1 follower"},
		{spotify.Item{Type: "episode", Name: "178: Ubiquiti", Show: "Darknet Diaries", DurationMS: 2429088}, "178: Ubiquiti — Darknet Diaries · 40m29s"},
	}
	for _, test := range tests {
		if got := itemHuman(ctx.Output, test.item); got != test.want {
			t.Fatalf("item=%#v line=%q want=%q", test.item, got, test.want)
		}
	}
}

func TestPlaybackHumanOmitsUnknownFields(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatHuman)
	if got := playbackHuman(ctx.Output, spotify.PlaybackStatus{}); got != "PAUSED" {
		t.Fatalf("playback=%q", got)
	}
	if got := playbackHuman(ctx.Output, spotify.PlaybackStatus{IsPlaying: true, Item: &spotify.Item{Name: "Song"}}); got != "PLAYING Song" {
		t.Fatalf("playback=%q", got)
	}
}

func TestItemPlainPreservesUnknownFieldPositions(t *testing.T) {
	tests := []struct {
		item spotify.Item
		want string
	}{
		{spotify.Item{ID: "a1", Type: "album", Name: "Album"}, "album\ta1\tAlbum\t\t\t0"},
		{spotify.Item{ID: "ar1", Type: "artist", Name: "Artist"}, "artist\tar1\tArtist\t0"},
		{spotify.Item{ID: "p1", Type: "playlist", Name: "Playlist"}, "playlist\tp1\tPlaylist\t\t0"},
		{spotify.Item{ID: "s1", Type: "show", Name: "Show"}, "show\ts1\tShow\t\t0"},
		{spotify.Item{ID: "e1", Type: "episode", Name: "Episode"}, "episode\te1\tEpisode\t0"},
	}
	for _, test := range tests {
		if got := itemPlain(test.item); got != test.want {
			t.Fatalf("item=%#v line=%q want=%q", test.item, got, test.want)
		}
	}
}

func TestFormatThousands(t *testing.T) {
	for value, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 5567807: "5,567,807"} {
		if got := formatThousands(value); got != want {
			t.Fatalf("formatThousands(%d)=%q want=%q", value, got, want)
		}
	}
}

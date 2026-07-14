package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestPlayCmdURI(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		PlayFn: func(ctx context.Context, uri string) error {
			if uri != "spotify:track:t1" {
				t.Fatalf("uri %s", uri)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlayCmd{Items: []string{"spotify:track:t1"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestPlayCmdInvalidResource(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(&testutil.SpotifyMock{PlayFn: func(ctx context.Context, uri string) error { return nil }})
	cmd := PlayCmd{Items: []string{"spotify:bad:t1"}}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPauseNextPrevErrors(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		PauseFn:    func(ctx context.Context) error { return errors.New("pause") },
		NextFn:     func(ctx context.Context) error { return errors.New("next") },
		PreviousFn: func(ctx context.Context) error { return errors.New("prev") },
	}
	ctx.SetSpotify(mock)
	if err := (&PauseCmd{}).Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
	if err := (&NextCmd{}).Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
	if err := (&PrevCmd{}).Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSeekCmdInvalidPosition(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		SeekFn: func(ctx context.Context, position int) error {
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := SeekCmd{Position: "bad"}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVolumeCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		VolumeFn: func(ctx context.Context, volume int) error {
			return errors.New("boom")
		},
	}
	ctx.SetSpotify(mock)
	cmd := VolumeCmd{Level: 25}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestShuffleCmdInvalid(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(&testutil.SpotifyMock{ShuffleFn: func(ctx context.Context, enabled bool) error { return nil }})
	cmd := ShuffleCmd{State: "maybe"}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestShuffleCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(&testutil.SpotifyMock{ShuffleFn: func(ctx context.Context, enabled bool) error { return errors.New("boom") }})
	cmd := ShuffleCmd{State: "on"}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRepeatCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(&testutil.SpotifyMock{RepeatFn: func(ctx context.Context, mode string) error { return errors.New("boom") }})
	cmd := RepeatCmd{Mode: "off"}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestStatusCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(&testutil.SpotifyMock{PlaybackFn: func(ctx context.Context) (spotify.PlaybackStatus, error) {
		return spotify.PlaybackStatus{}, errors.New("boom")
	}})
	cmd := StatusCmd{}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSeekCmdError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(&testutil.SpotifyMock{SeekFn: func(ctx context.Context, position int) error { return errors.New("boom") }})
	cmd := SeekCmd{Position: "1000"}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParsePositionMilliseconds(t *testing.T) {
	ms, err := parsePosition("120000")
	if err != nil || ms != 120000 {
		t.Fatalf("parse: %v %d", err, ms)
	}
	if _, err := parsePosition("1:xx"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseToggleOff(t *testing.T) {
	if v, _ := parseToggle("off"); v {
		t.Fatalf("expected false")
	}
}

func TestPlayCmdShuffleError(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	ctx.SetSpotify(&testutil.SpotifyMock{
		ShuffleFn: func(ctx context.Context, enabled bool) error { return errors.New("boom") },
		PlayFn:    func(ctx context.Context, uri string) error { t.Fatalf("play should not run"); return nil },
	})
	cmd := PlayCmd{Items: []string{"spotify:track:t1"}, Shuffle: true}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlayCmdMultiTrackOrdered(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	var got []string
	mock := &testutil.SpotifyMock{
		PlayTracksFn: func(ctx context.Context, uris []string) error {
			got = uris
			return nil
		},
		PlayFn: func(context.Context, string) error { t.Fatalf("Play should not run for multi-track"); return nil },
	}
	ctx.SetSpotify(mock)
	cmd := PlayCmd{Items: []string{"spotify:track:t1", "spotify:track:t2", "spotify:track:t3"}}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := []string{"spotify:track:t1", "spotify:track:t2", "spotify:track:t3"}
	if len(got) != len(want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("uri[%d]=%s want %s (order matters)", i, got[i], want[i])
		}
	}
}

func TestPlayCmdMultiTrackRawIDsRequireTrackType(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	var got []string
	mock := &testutil.SpotifyMock{
		PlayTracksFn: func(ctx context.Context, uris []string) error {
			got = uris
			return nil
		},
	}
	ctx.SetSpotify(mock)

	// Raw IDs without --type are rejected.
	if err := (&PlayCmd{Items: []string{"id1", "id2"}}).Run(ctx); err == nil {
		t.Fatalf("expected error for raw ids without type")
	}
	// Raw IDs with --type track build ordered track URIs.
	if err := (&PlayCmd{Items: []string{"id1", "id2"}, Type: "track"}).Run(ctx); err != nil {
		t.Fatalf("run with type track: %v", err)
	}
	if len(got) != 2 || got[0] != "spotify:track:id1" || got[1] != "spotify:track:id2" {
		t.Fatalf("unexpected uris: %#v", got)
	}
	// Raw IDs with a non-track --type are rejected.
	if err := (&PlayCmd{Items: []string{"id1", "id2"}, Type: "album"}).Run(ctx); err == nil {
		t.Fatalf("expected error for non-track type in multi-track mode")
	}
}

func TestPlayCmdMultiTrackRejectsNonTrack(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		PlayTracksFn: func(context.Context, []string) error {
			t.Fatalf("PlayTracks should not run when a non-track item is present")
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlayCmd{Items: []string{"spotify:track:t1", "spotify:album:a1"}}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error for album in multi-track mode")
	}
}

func TestPlayCmdMultiTrackShuffle(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	calls := []string{}
	mock := &testutil.SpotifyMock{
		ShuffleFn: func(ctx context.Context, enabled bool) error {
			if !enabled {
				t.Fatalf("expected shuffle enabled")
			}
			calls = append(calls, "shuffle")
			return nil
		},
		PlayTracksFn: func(ctx context.Context, uris []string) error {
			calls = append(calls, "playtracks")
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlayCmd{Items: []string{"spotify:track:t1", "spotify:track:t2"}, Shuffle: true}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(calls) != 2 || calls[0] != "shuffle" || calls[1] != "playtracks" {
		t.Fatalf("unexpected call order: %#v", calls)
	}
}

package cli

import (
	"context"
	"testing"

	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestStatusCmd(t *testing.T) {
	ctx, out, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		PlaybackFn: func(ctx context.Context) (spotify.PlaybackStatus, error) {
			return spotify.PlaybackStatus{IsPlaying: true, ProgressMS: 1000, Device: spotify.Device{Name: "Desk"}}, nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := StatusCmd{}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() == "" {
		t.Fatalf("expected output")
	}
}

func TestPlayCmdWithType(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		PlayFn: func(ctx context.Context, uri string) error {
			if uri != "spotify:track:abc" {
				t.Fatalf("uri %s", uri)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlayCmd{Items: []string{"abc"}, Type: "track"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestPlayCmdShuffle(t *testing.T) {
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
		PlayFn: func(ctx context.Context, uri string) error {
			if uri != "spotify:track:abc" {
				t.Fatalf("uri %s", uri)
			}
			calls = append(calls, "play")
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := PlayCmd{Items: []string{"abc"}, Type: "track", Shuffle: true}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(calls) != 2 || calls[0] != "shuffle" || calls[1] != "play" {
		t.Fatalf("unexpected call order: %#v", calls)
	}
}

func TestVolumeCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		VolumeFn: func(ctx context.Context, volume int) error {
			if volume != 50 {
				t.Fatalf("volume %d", volume)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := VolumeCmd{Level: 50}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestShuffleCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		ShuffleFn: func(ctx context.Context, enabled bool) error {
			if !enabled {
				t.Fatalf("expected true")
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := ShuffleCmd{State: "on"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRepeatCmd(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		RepeatFn: func(ctx context.Context, mode string) error {
			if mode != "track" {
				t.Fatalf("mode %s", mode)
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	cmd := RepeatCmd{Mode: "track"}
	if err := cmd.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestPauseNextPrevSeek(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{
		PauseFn:    func(ctx context.Context) error { return nil },
		NextFn:     func(ctx context.Context) error { return nil },
		PreviousFn: func(ctx context.Context) error { return nil },
		SeekFn: func(ctx context.Context, position int) error {
			if position == 0 {
				t.Fatalf("position")
			}
			return nil
		},
	}
	ctx.SetSpotify(mock)
	if err := (&PauseCmd{}).Run(ctx); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if err := (&NextCmd{}).Run(ctx); err != nil {
		t.Fatalf("next: %v", err)
	}
	if err := (&PrevCmd{}).Run(ctx); err != nil {
		t.Fatalf("prev: %v", err)
	}
	if err := (&SeekCmd{Position: "1:00"}).Run(ctx); err != nil {
		t.Fatalf("seek: %v", err)
	}
}

func TestParsePosition(t *testing.T) {
	ms, err := parsePosition("2:03")
	if err != nil || ms <= 0 {
		t.Fatalf("parse: %v", err)
	}
	if _, err := parsePosition(""); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := parsePosition("1:2:3"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseToggle(t *testing.T) {
	if v, _ := parseToggle("on"); !v {
		t.Fatalf("expected true")
	}
	if _, err := parseToggle("maybe"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPlayCmdMissingType(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	mock := &testutil.SpotifyMock{PlayFn: func(ctx context.Context, uri string) error { return nil }}
	ctx.SetSpotify(mock)
	cmd := PlayCmd{Items: []string{"abc"}}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVolumeCmdInvalid(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	cmd := VolumeCmd{Level: 200}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRepeatCmdInvalid(t *testing.T) {
	ctx, _, _ := testutil.NewTestContext(t, output.FormatPlain)
	cmd := RepeatCmd{Mode: "bad"}
	if err := cmd.Run(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestOnOff(t *testing.T) {
	if onOff(false) != "off" {
		t.Fatalf("expected off")
	}
}

package cli

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
	"github.com/steipete/spogo/internal/testutil"
)

func TestSeekRejectsInvalidPositionBeforePlayback(t *testing.T) {
	for _, input := range []string{"-1", "-1:30", "1:-30", "0:60", "1:99", fmt.Sprintf("%d:00", math.MaxInt/60000+1), strconv.Itoa(math.MaxInt) + "0"} {
		t.Run(input, func(t *testing.T) {
			ctx, out, _ := testutil.NewTestContext(t, output.FormatJSON)
			ctx.SetSpotify(&testutil.SpotifyMock{SeekFn: func(context.Context, int) error {
				t.Error("invalid input reached playback")
				return nil
			}})
			if err := (&SeekCmd{Position: input}).Run(ctx); err == nil {
				t.Fatal("invalid position reported success")
			}
			if out.Len() != 0 {
				t.Fatalf("unexpected success output: %s", out)
			}
		})
	}
}

func TestParsePositionWithoutDurationOverflow(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"0", 0},
		{"0:00", 0},
		{"2:03", 123000},
		{" 90000 ", 90000},
		{strconv.Itoa(math.MaxInt), math.MaxInt},
		{fmt.Sprintf("%d:%02d", math.MaxInt/60000, (math.MaxInt%60000)/1000), math.MaxInt / 1000 * 1000},
	}
	if strconv.IntSize == 64 {
		large, err := strconv.Atoi("9223372080000")
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, struct {
			input string
			want  int
		}{"153722868:00", large})
		if got := humanDuration(large); got != "2562047h48m00s" {
			t.Errorf("duration = %q", got)
		}
	}
	for _, tc := range cases {
		if got, err := parsePosition(tc.input); err != nil || got != tc.want {
			t.Errorf("parsePosition(%q) = %d, %v; want %d", tc.input, got, err, tc.want)
		}
	}
}

func TestTopTrackRanksIncludePageOffset(t *testing.T) {
	for _, format := range []output.Format{output.FormatPlain, output.FormatHuman} {
		t.Run(fmt.Sprint(format), func(t *testing.T) {
			ctx, out, _ := testutil.NewTestContext(t, format)
			ctx.SetSpotify(&testutil.SpotifyMock{GetUsersTopTracksFn: func(_ context.Context, _ string, limit, offset int) (spotify.TopTracksResult, error) {
				if limit != 2 || offset != 20 {
					t.Fatalf("request: limit=%d offset=%d", limit, offset)
				}
				return spotify.TopTracksResult{Total: 40, Limit: 2, Offset: 20, Items: []spotify.Item{{ID: "t21", Type: "track", Name: "Twenty-one"}, {ID: "t22", Type: "track", Name: "Twenty-two"}}}, nil
			}})
			if err := (&UserTopTracksCmd{Period: "long_term", Limit: 2, Offset: 20}).Run(ctx); err != nil {
				t.Fatal(err)
			}
			for _, rank := range []string{"21", "22"} {
				prefix := rank + "\ttrack\t"
				if format == output.FormatHuman {
					prefix = rank + ". "
				}
				if !strings.Contains(out.String(), prefix) {
					t.Errorf("missing rank %s: %s", rank, out)
				}
			}
		})
	}
}

package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/steipete/spogo/internal/app"
	"github.com/steipete/spogo/internal/output"
	"github.com/steipete/spogo/internal/spotify"
)

const (
	maxHistoryItems = 200
	maxHistoryPage  = 50
	userPeriodList  = "long_term, medium_term, short_term"
)

type UserCmd struct {
	TopTracks UserTopTracksCmd `kong:"cmd,help='Show your top tracks by affinity ranking.'"`
	History   UserHistoryCmd   `kong:"cmd,help='Show recently played tracks available from Spotify.'"`
}

type UserTopTracksCmd struct {
	Period string `help:"Time range: long_term (years), medium_term (~6 months), short_term (~4 weeks)." default:"long_term"`
	Limit  int    `help:"Limit results." default:"20"`
	Offset int    `help:"Offset results." default:"0"`
}

type UserHistoryCmd struct {
	Period string `help:"History window: long_term, medium_term, short_term." default:"long_term"`
	Limit  int    `help:"Limit results (max 200)." default:"20"`
	After  int64  `help:"Lower-bound filter, Unix timestamp (ms)."`
	Before int64  `help:"Return items played before this Unix timestamp (ms)."`
}

var validUserPeriods = map[string]bool{
	"long_term":   true,
	"medium_term": true,
	"short_term":  true,
}

func topTracksTimeRange(period string) string {
	if validUserPeriods[period] {
		return period
	}
	return ""
}

var historyPeriodDurations = map[string]time.Duration{
	"medium_term": 182 * 24 * time.Hour,
	"short_term":  28 * 24 * time.Hour,
}

func historyAfter(period string) int64 {
	d, ok := historyPeriodDurations[period]
	if !ok {
		return 0
	}
	return time.Now().Add(-d).UnixMilli()
}

func (cmd *UserTopTracksCmd) Run(ctx *app.Context) error {
	timeRange := topTracksTimeRange(cmd.Period)
	if timeRange == "" {
		return fmt.Errorf("invalid top-tracks period %q; use one of: %s", cmd.Period, userPeriodList)
	}
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}
	limit := clampLimit(cmd.Limit)
	res, err := client.GetUsersTopTracks(cmdCtx, timeRange, limit, cmd.Offset)
	if err != nil {
		return err
	}
	plain, human := renderTopTracks(ctx.Output, res.Items)
	payload := map[string]any{
		"total":      res.Total,
		"limit":      res.Limit,
		"offset":     res.Offset,
		"items":      res.Items,
		"time_range": timeRange,
		"period":     cmd.Period,
	}
	if ctx.Output.Format == output.FormatHuman {
		header := fmt.Sprintf("Top tracks (%s): %d", timeRange, res.Total)
		human = append([]string{header}, human...)
	}
	return ctx.Output.Emit(payload, plain, human)
}

func (cmd *UserHistoryCmd) Run(ctx *app.Context) error {
	if !validUserPeriods[cmd.Period] {
		return fmt.Errorf("invalid history period %q; use one of: %s", cmd.Period, userPeriodList)
	}
	client, cmdCtx, err := spotifyClient(ctx)
	if err != nil {
		return err
	}

	limit := clampHistoryLimit(cmd.Limit)
	after := historyAfter(cmd.Period)
	if cmd.After > after {
		after = cmd.After
	}

	var allItems []spotify.RecentlyPlayedItem
	var lastCursors *spotify.Cursors
	before := cmd.Before
	stopAtLowerBound := false

	for {
		remaining := limit - len(allItems)
		if remaining <= 0 {
			break
		}
		pageLimit := remaining
		if pageLimit > maxHistoryPage {
			pageLimit = maxHistoryPage
		}

		res, err := client.GetRecentlyPlayed(cmdCtx, pageLimit, 0, before)
		if err != nil {
			return err
		}
		if res.Cursors != nil {
			lastCursors = res.Cursors
		}
		for _, item := range res.Items {
			ms, err := parseRFC3339Milli(item.PlayedAt)
			if err != nil {
				return fmt.Errorf("invalid played_at %q: %w", item.PlayedAt, err)
			}
			if cmd.Before > 0 && ms >= cmd.Before {
				continue
			}
			if after > 0 && ms <= after {
				stopAtLowerBound = true
				continue
			}
			allItems = append(allItems, item)
			if len(allItems) >= limit {
				break
			}
		}
		if len(res.Items) == 0 || res.Cursors == nil || res.Cursors.Before == "" || len(allItems) >= limit || stopAtLowerBound {
			break
		}
		nextBefore, err := strconv.ParseInt(res.Cursors.Before, 10, 64)
		if err != nil || nextBefore == before {
			break
		}
		before = nextBefore
	}

	plain, human := renderRecentlyPlayed(ctx.Output, allItems)
	payload := map[string]any{
		"items":                 allItems,
		"cursors":               lastCursors,
		"total_fetched":         len(allItems),
		"limit":                 limit,
		"max_allowed":           maxHistoryItems,
		"period":                cmd.Period,
		"requested_after":       cmd.After,
		"requested_before":      cmd.Before,
		"effective_lower_bound": after,
	}
	if ctx.Output.Format == output.FormatHuman {
		header := fmt.Sprintf("Recently played (%s): %d", cmd.Period, len(allItems))
		human = append([]string{header}, human...)
	}
	return ctx.Output.Emit(payload, plain, human)
}

func clampHistoryLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > maxHistoryItems {
		return maxHistoryItems
	}
	return limit
}

func parseRFC3339Milli(s string) (int64, error) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return 0, err
	}
	return t.UnixMilli(), nil
}

func renderTopTracks(w *output.Writer, items []spotify.Item) (plain []string, human []string) {
	accent := w.Theme.Accent
	muted := w.Theme.Muted
	plain = make([]string, 0, len(items))
	human = make([]string, 0, len(items))
	for i, item := range items {
		rank := i + 1
		plain = append(plain, fmt.Sprintf("%d\ttrack\t%s\t%s\t%s\t%s\t%s", rank, item.ID, item.Name, strings.Join(item.Artists, ", "), item.Album, item.URI))
		human = append(human, fmt.Sprintf("%d. %s — %s %s", rank, accent(item.Name), strings.Join(item.Artists, ", "), muted("· "+item.Album)))
	}
	return plain, human
}

func renderRecentlyPlayed(w *output.Writer, items []spotify.RecentlyPlayedItem) (plain []string, human []string) {
	accent := w.Theme.Accent
	muted := w.Theme.Muted
	plain = make([]string, 0, len(items))
	human = make([]string, 0, len(items))
	for _, item := range items {
		plain = append(plain, fmt.Sprintf("%s\ttrack\t%s\t%s\t%s\t%s\t%s", item.PlayedAt, item.Track.ID, item.Track.Name, strings.Join(item.Track.Artists, ", "), item.Track.Album, item.Track.URI))
		human = append(human, fmt.Sprintf("%s — %s %s", accent(item.Track.Name), strings.Join(item.Track.Artists, ", "), muted("· "+item.PlayedAt)))
	}
	return plain, human
}

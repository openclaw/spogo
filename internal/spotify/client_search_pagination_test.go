package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

// searchPages serves "total" tracks in the requested slices and records each
// "limit/offset" pair the client asked for.
func searchPages(t *testing.T, total int) (http.HandlerFunc, *[]string) {
	t.Helper()
	calls := make([]string, 0, 4)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		calls = append(calls, fmt.Sprintf("%d/%d", limit, offset))
		items := make([]json.RawMessage, 0, limit)
		for i := offset; i < offset+limit && i < total; i++ {
			items = append(items, json.RawMessage(fmt.Sprintf("{\"id\":\"t%d\",\"name\":\"Track %d\"}", i, i)))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tracks": map[string]any{"items": items, "limit": limit, "offset": offset, "total": total},
		})
	})
	return handler, &calls
}

func TestSearchPagesAboveDevelopmentLimit(t *testing.T) {
	handler, calls := searchPages(t, 25)
	client, closeFn := newTestClient(t, handler)
	defer closeFn()

	res, err := client.Search(context.Background(), "track", "song", 25, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if got := fmt.Sprint(*calls); got != "[10/0 10/10 5/20]" {
		t.Fatalf("requests = %s, want three pages capped at ten", got)
	}
	if len(res.Items) != 25 || res.Total != 25 || res.Limit != 25 || res.Offset != 0 {
		t.Fatalf("unexpected result: items=%d total=%d limit=%d offset=%d", len(res.Items), res.Total, res.Limit, res.Offset)
	}
}

func TestSearchSingleRequestWithinLimit(t *testing.T) {
	handler, calls := searchPages(t, 40)
	client, closeFn := newTestClient(t, handler)
	defer closeFn()

	res, err := client.Search(context.Background(), "track", "song", 5, 2)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if got := fmt.Sprint(*calls); got != "[5/2]" {
		t.Fatalf("requests = %s, want one request", got)
	}
	if len(res.Items) != 5 || res.Offset != 2 {
		t.Fatalf("unexpected result: items=%d offset=%d", len(res.Items), res.Offset)
	}
}

func TestSearchStopsWhenResultsRunOut(t *testing.T) {
	handler, calls := searchPages(t, 12)
	client, closeFn := newTestClient(t, handler)
	defer closeFn()

	res, err := client.Search(context.Background(), "track", "song", 30, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if got := fmt.Sprint(*calls); got != "[10/0 10/10]" {
		t.Fatalf("requests = %s, want to stop after the short page", got)
	}
	if len(res.Items) != 12 {
		t.Fatalf("items = %d, want every available result", len(res.Items))
	}
}

func TestSearchPagesFromOffset(t *testing.T) {
	handler, calls := searchPages(t, 40)
	client, closeFn := newTestClient(t, handler)
	defer closeFn()

	res, err := client.Search(context.Background(), "track", "song", 15, 7)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if got := fmt.Sprint(*calls); got != "[10/7 5/17]" {
		t.Fatalf("requests = %s, want pages continuing from the requested offset", got)
	}
	if len(res.Items) != 15 || res.Items[0].ID != "t7" {
		t.Fatalf("unexpected result: items=%d first=%q", len(res.Items), res.Items[0].ID)
	}
}

package spotify

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestLibraryMutationsBatchAtFortyURIs(t *testing.T) {
	for _, kind := range []string{"track", "album", "artist"} {
		for _, method := range []string{http.MethodPut, http.MethodDelete} {
			t.Run(kind+"/"+method, func(t *testing.T) {
				var sizes []int
				var received []string
				client, closeFn := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/me/library" || r.Method != method {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					}
					uris := strings.Split(r.URL.Query().Get("uris"), ",")
					sizes = append(sizes, len(uris))
					received = append(received, uris...)
					if len(uris) > 40 {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					w.WriteHeader(http.StatusNoContent)
				}))
				defer closeFn()
				ids := make([]string, 85)
				want := make([]string, len(ids))
				for i := range ids {
					ids[i] = fmt.Sprintf("id%d", i)
					want[i] = "spotify:" + kind + ":" + ids[i]
				}
				var err error
				if kind == "artist" {
					err = client.FollowArtists(context.Background(), ids, method)
				} else {
					err = client.LibraryModify(context.Background(), kind, ids, method)
				}
				if err != nil {
					t.Fatalf("mutation: %v", err)
				}
				if !reflect.DeepEqual(sizes, []int{40, 40, 5}) || !reflect.DeepEqual(received, want) {
					t.Fatalf("batches = %v, URIs = %v", sizes, received)
				}
			})
		}
	}
}

func TestLibraryMutationStopsAfterFailedBatch(t *testing.T) {
	calls := 0
	client, closeFn := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 2 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer closeFn()
	ids := make([]string, 85)
	for i := range ids {
		ids[i] = fmt.Sprintf("id%d", i)
	}
	if err := client.LibraryModify(context.Background(), "track", ids, http.MethodPut); err == nil {
		t.Fatal("expected second batch error")
	}
	if calls != 2 {
		t.Fatalf("requests = %d, want 2", calls)
	}
}

func TestLibraryMutationRejectsEmptyInput(t *testing.T) {
	client, closeFn := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("empty mutation must not send an HTTP request")
	}))
	defer closeFn()
	if err := client.LibraryModify(context.Background(), "track", nil, http.MethodPut); err == nil {
		t.Fatal("expected empty-input error")
	}
}

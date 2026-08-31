package spotify

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
)

func TestSearchDocumentedResponse(t *testing.T) {
	// Synthetic catalog data using Spotify's documented search response shape:
	// https://developer.spotify.com/documentation/web-api/reference/search
	fixture, err := os.ReadFile("testdata/search_response.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"track", "album", "artist", "playlist", "show", "episode"} {
		for _, engine := range []string{"web", "connect-web-fallback"} {
			t.Run(kind+"/"+engine, func(t *testing.T) {
				transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodGet || req.URL.Path != "/v1/search" {
						t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
					}
					params := req.URL.Query()
					if params.Get("type") != kind || params.Get("q") != "fixture" || params.Get("limit") != "1" || params.Get("offset") != "1" {
						t.Errorf("unexpected search parameters: %v", params)
					}
					return &http.Response{
						StatusCode:    http.StatusOK,
						Header:        http.Header{"Content-Type": []string{"application/json"}},
						ContentLength: int64(len(fixture)),
						Body:          io.NopCloser(bytes.NewReader(fixture)),
					}, nil
				})
				client, err := NewClient(Options{
					TokenProvider: staticTokenProvider{},
					HTTPClient:    &http.Client{Transport: transport},
				})
				if err != nil {
					t.Fatal(err)
				}
				search := client.Search
				if engine == "connect-web-fallback" {
					search = newConnectClientForTests(transport).searchViaWebAPI
				}
				result, err := search(context.Background(), kind, "fixture", 1, 1)
				if err != nil {
					t.Fatalf("search: %v", err)
				}
				if result.Type != kind || result.Limit != 1 || result.Offset != 1 || result.Total != 3 {
					t.Errorf("unexpected result metadata: %#v", result)
				}
				if len(result.Items) != 1 {
					t.Fatalf("expected one item, got %d", len(result.Items))
				}
				item := result.Items[0]
				if item.Type != kind || item.ID != kind+"1" || item.Name != "Fixture "+kind || item.URI != "spotify:"+kind+":"+kind+"1" {
					t.Errorf("unexpected item: %#v", item)
				}
				if kind == "track" && (item.Album != "Fixture album" || len(item.Artists) != 1 || item.Artists[0] != "Fixture artist" || item.DurationMS != 123000) {
					t.Errorf("unexpected track metadata: %#v", item)
				}
			})
		}
	}
}

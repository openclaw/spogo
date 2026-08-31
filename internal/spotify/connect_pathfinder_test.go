package spotify

import (
	"context"
	"net/http"
	"testing"
)

func TestPathfinderSearch(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		payload := map[string]any{
			"data": map[string]any{
				"searchV2": map[string]any{
					"tracksV2": map[string]any{
						"totalCount": 1,
						"items": []any{
							map[string]any{
								"uri":  "spotify:track:abc",
								"name": "Song",
								"artists": []any{
									map[string]any{"name": "Artist"},
								},
							},
						},
					},
				},
			},
		}
		return jsonResponse(http.StatusOK, payload), nil
	})
	client := newConnectClientForTests(transport)
	client.hashes.hashes["searchDesktop"] = "hash"
	result, err := client.Search(context.Background(), "track", "song", 1, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Name != "Song" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestPathfinderError(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		payload := map[string]any{
			"errors": []any{map[string]any{"message": "bad"}},
		}
		return jsonResponse(http.StatusOK, payload), nil
	})
	client := newConnectClientForTests(transport)
	client.hashes.hashes["searchDesktop"] = "hash"
	_, err := client.Search(context.Background(), "track", "song", 1, 0)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPathfinderErrorMissingMessage(t *testing.T) {
	err := pathfinderError(map[string]any{"errors": []any{map[string]any{}}})
	if err == nil {
		t.Fatalf("expected error")
	}
	if err.Error() != "pathfinder error" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPathfinderEmptyQuery(t *testing.T) {
	client := newConnectClientForTests(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]any{}), nil
	}))
	if _, err := client.Search(context.Background(), "track", "  ", 1, 0); err == nil {
		t.Fatalf("expected error")
	}
}

func TestInfoByOperationMissing(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]any{"data": map[string]any{}}), nil
	})
	client := newConnectClientForTests(transport)
	client.hashes.hashes["getTrack"] = "hash"
	if _, err := client.infoByOperation(context.Background(), "getTrack", map[string]any{}, "track"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPathfinderErrorEmpty(t *testing.T) {
	if err := pathfinderError(map[string]any{"errors": []any{}}); err != nil {
		t.Fatalf("unexpected error")
	}
	if err := pathfinderError(map[string]any{"errors": "bad"}); err != nil {
		t.Fatalf("unexpected error")
	}
}

func TestGraphQLNilVariables(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, map[string]any{"data": map[string]any{}}), nil
	})
	client := newConnectClientForTests(transport)
	client.language = "en-US"
	client.hashes.hashes["searchDesktop"] = "hash"
	if _, err := client.graphQL(context.Background(), "searchDesktop", nil); err != nil {
		t.Fatalf("graphQL: %v", err)
	}
}

func TestGraphQLHTTPError(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return textResponse(http.StatusInternalServerError, "fail"), nil
	})
	client := newConnectClientForTests(transport)
	client.hashes.hashes["searchDesktop"] = "hash"
	if _, err := client.graphQL(context.Background(), "searchDesktop", map[string]any{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGraphQLRequiresInitializedClient(t *testing.T) {
	if _, err := (&ConnectClient{}).graphQL(context.Background(), "searchDesktop", map[string]any{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPathfinderFallbackToWeb(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host == "api-partner.spotify.com" {
			return textResponse(http.StatusInternalServerError, "fail"), nil
		}
		payload := map[string]any{
			"tracks": map[string]any{
				"items": []map[string]any{{
					"id":   "t1",
					"uri":  "spotify:track:t1",
					"name": "Song",
				}},
				"limit":  1,
				"offset": 0,
				"total":  1,
			},
		}
		return jsonResponse(http.StatusOK, payload), nil
	})
	client := newConnectClientForTests(transport)
	client.hashes.hashes["searchDesktop"] = "hash"
	client.searchURL = "https://search.local/search"

	result, err := client.Search(context.Background(), "track", "song", 1, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != "t1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestSearchViaWebAPIDefaultClient(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		payload := map[string]any{
			"tracks": map[string]any{
				"items": []map[string]any{{
					"id":   "t1",
					"uri":  "spotify:track:t1",
					"name": "Song",
				}},
				"limit":  1,
				"offset": 0,
				"total":  1,
			},
		}
		return jsonResponse(http.StatusOK, payload), nil
	})
	client := newConnectClientForTests(transport)
	client.searchURL = ""
	client.searchClient = nil

	result, err := client.searchViaWebAPI(context.Background(), "track", "song", 1, 0)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != "t1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestSearchViaWebAPIMissingKind(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		payload := map[string]any{"album": map[string]any{}}
		return jsonResponse(http.StatusOK, payload), nil
	})
	client := newConnectClientForTests(transport)
	client.searchURL = "https://search.local/search"

	if _, err := client.searchViaWebAPI(context.Background(), "track", "song", 1, 0); err == nil {
		t.Fatalf("expected error")
	}
}

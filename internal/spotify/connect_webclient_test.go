package spotify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConnectWebClientCaches(t *testing.T) {
	client := &ConnectClient{
		source: cookieSourceStub{cookies: []*http.Cookie{{Name: "sp_dc", Value: "token"}}},
		client: &http.Client{},
		market: "US",
	}
	first, err := client.webClient()
	if err != nil {
		t.Fatalf("web client: %v", err)
	}
	second, err := client.webClient()
	if err != nil {
		t.Fatalf("web client again: %v", err)
	}
	if first != second {
		t.Fatalf("expected cached web client")
	}
}

func TestConnectWebClientUsesInjectedClient(t *testing.T) {
	web, err := NewClient(Options{TokenProvider: staticTokenProvider{}})
	if err != nil {
		t.Fatalf("new web client: %v", err)
	}
	connect, err := NewConnectClient(ConnectOptions{
		Source:    cookieSourceStub{cookies: []*http.Cookie{{Name: "sp_dc", Value: "cookie"}}},
		WebClient: web,
	})
	if err != nil {
		t.Fatalf("new connect client: %v", err)
	}
	got, err := connect.webClient()
	if err != nil {
		t.Fatalf("web client: %v", err)
	}
	if got != web {
		t.Fatalf("expected injected Web API client")
	}
}

func TestConnectSearchViaWebAPIUsesInjectedClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("authorization header = %q", got)
		}
		_, _ = w.Write([]byte(`{"tracks":{"items":[{"id":"t1","uri":"spotify:track:t1","name":"Song"}],"limit":1,"offset":0,"total":1}}`))
	}))
	t.Cleanup(server.Close)
	web, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new web client: %v", err)
	}
	connect := &ConnectClient{web: web}

	result, err := connect.searchViaWebAPI(context.Background(), "track", "song", 1, 0)
	if err != nil {
		t.Fatalf("search via injected web client: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != "t1" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

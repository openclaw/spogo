package spotify

import (
	"net/http"
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

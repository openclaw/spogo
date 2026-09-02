package spotify

import (
	"net/http"
	"time"
)

func newConnectClientForTests(transport http.RoundTripper) *ConnectClient {
	client := &http.Client{Transport: transport}
	web, err := NewClient(Options{TokenProvider: staticTokenProvider{}, HTTPClient: client})
	if err != nil {
		panic(err)
	}
	session := &connectSession{
		client:       client,
		token:        Token{AccessToken: "access", ExpiresAt: time.Now().Add(time.Hour), ClientID: "client"},
		clientToken:  "client-token",
		clientTokenT: time.Now().Add(time.Hour),
		clientVer:    "1.0.0",
		deviceID:     "device",
	}
	hashes := &hashResolver{client: client, session: session, hashes: map[string]string{}}
	return &ConnectClient{client: client, session: session, hashes: hashes, web: web}
}

func newRegisteredConnectClientForTests(transport http.RoundTripper) *ConnectClient {
	client := newConnectClientForTests(transport)
	client.session.connectDeviceID = "device"
	client.session.connectionID = "conn"
	client.session.registeredAt = time.Now()
	return client
}

func attachConnectCacheForTests(client *ConnectClient, path string) {
	cache := newConnectCacheStore(path)
	client.cache = cache
	client.session.cache = cache
}

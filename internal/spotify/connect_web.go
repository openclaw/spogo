package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *ConnectClient) searchViaWebAPI(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	auth, err := c.session.auth(ctx)
	if err != nil {
		return SearchResult{}, err
	}
	params := url.Values{}
	params.Set("q", query)
	params.Set("type", kind)
	params.Set("limit", fmt.Sprint(limit))
	params.Set("offset", fmt.Sprint(offset))
	if c.market != "" && params.Get("market") == "" {
		params.Set("market", c.market)
	}
	if c.language != "" && params.Get("locale") == "" {
		params.Set("locale", c.language)
	}
	searchURL := c.searchURL
	if searchURL == "" {
		searchURL = "https://api.spotify.com/v1/search"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL+"?"+params.Encode(), nil)
	if err != nil {
		return SearchResult{}, err
	}
	applyRequestHeaders(req, requestHeaders{
		AccessToken:   auth.AccessToken,
		ClientToken:   auth.ClientToken,
		ClientVersion: auth.ClientVersion,
		Accept:        "application/json",
		Language:      c.language,
		AppPlatform:   defaultSpotifyAppPlatform,
	})
	resp, err := c.client.Do(req)
	if err != nil {
		return SearchResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return SearchResult{}, apiErrorFromResponse(resp)
	}
	var response map[string]searchContainer
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return SearchResult{}, err
	}
	container, ok := response[kind+"s"]
	if !ok {
		return SearchResult{}, fmt.Errorf("missing %s result", kind)
	}
	items := make([]Item, 0, len(container.Items))
	for _, raw := range container.Items {
		item, err := mapSearchItem(kind, raw)
		if err != nil {
			return SearchResult{}, err
		}
		items = append(items, item)
	}
	return SearchResult{
		Type:   kind,
		Limit:  container.Limit,
		Offset: container.Offset,
		Total:  container.Total,
		Items:  items,
	}, nil
}

func (c *ConnectClient) webClient() (*Client, error) {
	c.webMu.Lock()
	defer c.webMu.Unlock()
	if c.web != nil {
		return c.web, nil
	}
	client, err := NewClient(Options{
		TokenProvider: CookieTokenProvider{Source: c.source, Client: c.client},
		HTTPClient:    c.client,
		Market:        c.market,
		Language:      c.language,
		Device:        c.device,
	})
	if err != nil {
		return nil, err
	}
	c.web = client
	return client, nil
}

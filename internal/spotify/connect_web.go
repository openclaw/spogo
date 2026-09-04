package spotify

import (
	"context"
)

func (c *ConnectClient) searchViaWebAPI(ctx context.Context, kind, query string, limit, offset int) (SearchResult, error) {
	web, err := c.webClient()
	if err != nil {
		return SearchResult{}, err
	}
	return web.Search(ctx, kind, query, limit, offset)
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

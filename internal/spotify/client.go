package spotify

import (
	"errors"
	"net/http"
	"sync"
	"time"
)

const defaultHTTPClientTimeout = 10 * time.Second

type Options struct {
	TokenProvider TokenProvider
	HTTPClient    *http.Client
	BaseURL       string
	Market        string
	Language      string
	Device        string
	Timeout       time.Duration
}

type Client struct {
	baseURL  string
	market   string
	language string
	device   string
	client   *http.Client
	provider TokenProvider

	mu        sync.Mutex
	lastToken Token
}

func NewClient(opts Options) (*Client, error) {
	if opts.TokenProvider == nil {
		return nil, errors.New("token provider required")
	}
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = "https://api.spotify.com/v1"
	}
	client := opts.HTTPClient
	if client == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = defaultHTTPClientTimeout
		}
		client = &http.Client{Timeout: timeout}
	}
	return &Client{
		baseURL:  baseURL,
		market:   opts.Market,
		language: opts.Language,
		device:   opts.Device,
		client:   client,
		provider: opts.TokenProvider,
	}, nil
}

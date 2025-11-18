package proxy

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type Proxy struct {
	client *http.Client
}

func NewProxy() *Proxy {
	return &Proxy{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ExecuteRequest performs an HTTP GET request to the target URL
func (p *Proxy) ExecuteRequest(targetURL string) ([]byte, int, error) {
	resp, err := p.client.Get(targetURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp.StatusCode, nil
}

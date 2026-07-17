// Package elasticsearch contains searchable memory persistence owned by Go.
package elasticsearch

import (
	"agent-go/config"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	cfg  config.ElasticsearchConfig
	http *http.Client
}

func NewClient(cfg config.ElasticsearchConfig) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: config.Duration(cfg.Timeout, 10*time.Second)}}
}
func (c *Client) Do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.cfg.Addresses[0], "/")+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+c.cfg.APIKey)
	} else if c.cfg.Username != "" {
		req.SetBasicAuth(c.cfg.Username, c.cfg.Password)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Elasticsearch %s %s: %w", method, path, err)
	}
	return resp, nil
}
func Expect2xx(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("Elasticsearch returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
}

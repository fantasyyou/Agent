package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Memory struct {
	SessionID       string    `json:"session_id"`
	UserQuestion    string    `json:"user_question"`
	AssistantAnswer string    `json:"assistant_answer"`
	CreatedAt       time.Time `json:"created_at"`
}

type ESMemory struct {
	cfg    ElasticsearchConfig
	client *http.Client
}

func NewESMemory(cfg ElasticsearchConfig) *ESMemory {
	timeout, _ := parseDuration(cfg.Timeout, 10*time.Second)
	return &ESMemory{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

func (e *ESMemory) EnsureIndex(ctx context.Context) error {
	mapping := map[string]any{"mappings": map[string]any{
		"dynamic": "strict",
		"properties": map[string]any{
			"session_id":       map[string]any{"type": "keyword"},
			"user_question":    map[string]any{"type": "text"},
			"assistant_answer": map[string]any{"type": "text"},
			"created_at":       map[string]any{"type": "date"},
		},
	}}
	resp, err := e.do(ctx, http.MethodPut, "/"+url.PathEscape(e.cfg.Index), mapping)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest {
		var result struct {
			Error struct {
				Type string `json:"type"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&result)
		if result.Error.Type == "resource_already_exists_exception" {
			return nil
		}
	}
	return expect2xx(resp)
}

func (e *ESMemory) Store(ctx context.Context, memory Memory) error {
	path := "/" + url.PathEscape(e.cfg.Index) + "/_doc?refresh=wait_for"
	resp, err := e.do(ctx, http.MethodPost, path, memory)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return expect2xx(resp)
}

func (e *ESMemory) Search(ctx context.Context, sessionID, query string, limit int) ([]Memory, error) {
	body := map[string]any{
		"size": limit,
		"query": map[string]any{"bool": map[string]any{
			"filter": []any{map[string]any{"term": map[string]any{"session_id": sessionID}}},
			"should": []any{map[string]any{"multi_match": map[string]any{
				"query": query, "fields": []string{"user_question^2", "assistant_answer"},
			}}},
			"minimum_should_match": 0,
		}},
		"sort": []any{map[string]any{"_score": "desc"}, map[string]any{"created_at": "desc"}},
	}
	resp, err := e.do(ctx, http.MethodPost, "/"+url.PathEscape(e.cfg.Index)+"/_search", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := expect2xx(resp); err != nil {
		return nil, err
	}
	var result struct {
		Hits struct {
			Hits []struct {
				Source Memory `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Elasticsearch response: %w", err)
	}
	memories := make([]Memory, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		memories = append(memories, hit.Source)
	}
	return memories, nil
}

func (e *ESMemory) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	base := strings.TrimRight(e.cfg.Addresses[0], "/")
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if e.cfg.APIKey != "" {
		req.Header.Set("Authorization", "ApiKey "+e.cfg.APIKey)
	} else if e.cfg.Username != "" {
		req.SetBasicAuth(e.cfg.Username, e.cfg.Password)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Elasticsearch %s %s: %w", method, path, err)
	}
	return resp, nil
}

func expect2xx(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("Elasticsearch returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
}

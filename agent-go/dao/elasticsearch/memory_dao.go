package elasticsearch

import (
	"agent-go/model"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type MemoryDAO struct {
	client *Client
	index  string
}

func NewMemoryDAO(client *Client, index string) *MemoryDAO {
	return &MemoryDAO{client: client, index: index}
}
func (d *MemoryDAO) EnsureIndex(ctx context.Context) error {
	mapping := map[string]any{"mappings": map[string]any{"dynamic": "strict", "properties": map[string]any{"user_id": map[string]any{"type": "keyword"}, "session_id": map[string]any{"type": "keyword"}, "user_question": map[string]any{"type": "text"}, "assistant_answer": map[string]any{"type": "text"}, "created_at": map[string]any{"type": "date"}}}}
	resp, err := d.client.Do(ctx, http.MethodPut, "/"+url.PathEscape(d.index), mapping)
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
	return Expect2xx(resp)
}
func (d *MemoryDAO) Store(ctx context.Context, memory model.ConversationMemory) error {
	resp, err := d.client.Do(ctx, http.MethodPost, "/"+url.PathEscape(d.index)+"/_doc?refresh=wait_for", memory)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return Expect2xx(resp)
}
func (d *MemoryDAO) Search(ctx context.Context, userID, sessionID, query string, limit int) ([]model.ConversationMemory, error) {
	body := map[string]any{"size": limit, "query": map[string]any{"bool": map[string]any{"filter": []any{map[string]any{"term": map[string]any{"user_id": userID}}, map[string]any{"term": map[string]any{"session_id": sessionID}}}, "should": []any{map[string]any{"multi_match": map[string]any{"query": query, "fields": []string{"user_question^2", "assistant_answer"}}}}, "minimum_should_match": 0}}, "sort": []any{map[string]any{"_score": "desc"}, map[string]any{"created_at": "desc"}}}
	resp, err := d.client.Do(ctx, http.MethodPost, "/"+url.PathEscape(d.index)+"/_search", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := Expect2xx(resp); err != nil {
		return nil, err
	}
	var result struct {
		Hits struct {
			Hits []struct {
				Source model.ConversationMemory `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Elasticsearch response: %w", err)
	}
	memories := make([]model.ConversationMemory, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		memories = append(memories, hit.Source)
	}
	return memories, nil
}

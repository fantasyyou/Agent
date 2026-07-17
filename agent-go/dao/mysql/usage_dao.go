package mysql

import (
	"agent-go/model"
	"context"
	"database/sql"
	"fmt"
)

type UsageDAO struct{ db *sql.DB }

func NewUsageDAO(db *sql.DB) *UsageDAO { return &UsageDAO{db: db} }
func (d *UsageDAO) Create(ctx context.Context, usage model.ModelUsage) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO model_usages(id,user_id,session_id,request_id,provider,model,input_tokens,cached_tokens,output_tokens,total_tokens,total_cost,currency,latency_ms,status,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, usage.ID, usage.UserID, usage.SessionID, usage.RequestID, usage.Provider, usage.Model, usage.InputTokens, usage.CachedTokens, usage.OutputTokens, usage.TotalTokens, usage.TotalCost, usage.Currency, usage.LatencyMS, usage.Status, usage.CreatedAt)
	if err != nil {
		return fmt.Errorf("create model usage: %w", err)
	}
	return nil
}

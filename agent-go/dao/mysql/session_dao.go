package mysql

import (
	"agent-go/model"
	"context"
	"database/sql"
	"fmt"
)

type SessionDAO struct{ db *sql.DB }

func NewSessionDAO(db *sql.DB) *SessionDAO { return &SessionDAO{db: db} }
func (d *SessionDAO) Touch(ctx context.Context, session model.ConversationSession) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO conversation_sessions(id,user_id,status,created_at,last_active_at) VALUES(?,?,?,?,?) ON DUPLICATE KEY UPDATE last_active_at=VALUES(last_active_at),status=VALUES(status)`, session.ID, session.UserID, session.Status, session.CreatedAt, session.LastActiveAt)
	if err != nil {
		return fmt.Errorf("touch conversation session: %w", err)
	}
	return nil
}

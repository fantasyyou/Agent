package mysql

import (
	"agent-go/model"
	"context"
	"database/sql"
	"fmt"
)

type ActionExecutionDAO struct{ db *sql.DB }

func NewActionExecutionDAO(db *sql.DB) *ActionExecutionDAO { return &ActionExecutionDAO{db: db} }

func (d *ActionExecutionDAO) Create(ctx context.Context, execution model.ActionExecution) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO action_executions(id,user_id,session_id,request_id,intent,action,active_slot,status,result_message,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, execution.ID, execution.UserID, execution.SessionID, execution.RequestID, execution.Intent, execution.Action, execution.ActiveSlot, execution.Status, execution.ResultMessage, execution.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert action execution: %w", err)
	}
	return nil
}

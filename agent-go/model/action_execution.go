package model

import "time"

const (
	ActionExecutionStatusSuccess       = "success"
	ActionExecutionStatusNotConfigured = "not_configured"
)

// ActionExecution 记录 Go 对 Python 建议动作的实际执行结果。
type ActionExecution struct {
	ID            string
	UserID        string
	SessionID     string
	RequestID     string
	Intent        string
	Action        string
	ActiveSlot    string
	Status        string
	ResultMessage string
	CreatedAt     time.Time
}

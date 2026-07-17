package model

import "time"

// ConversationSession 表示持久化在 MySQL 中的一次客服会话及其归属关系。
type ConversationSession struct {
	// ID 是由网页客户端生成的会话唯一标识。
	ID string `json:"session_id"`
	// UserID 是拥有该会话的用户唯一标识。
	UserID string `json:"user_id"`
	// Status 是会话状态，可选值为 SessionStatusActive、SessionStatusClosed。
	Status string `json:"status"`
	// CreatedAt 是该会话收到第一条消息时的 UTC 时间。
	CreatedAt time.Time `json:"created_at"`
	// LastActiveAt 是该会话最近一次收到消息时的 UTC 时间。
	LastActiveAt time.Time `json:"last_active_at"`
}

package model

import "time"

// ConversationMemory 表示持久化在 Elasticsearch 中、可被全文检索的一轮问答记忆。
type ConversationMemory struct {
	// UserID 是记忆所属用户的唯一标识，也是用户数据隔离条件。
	UserID string `json:"user_id"`
	// SessionID 是产生该记忆的会话唯一标识。
	SessionID string `json:"session_id"`
	// UserQuestion 是用户的原始问题，用于全文检索和上下文还原。
	UserQuestion string `json:"user_question"`
	// AssistantAnswer 是客服对该问题的历史回答。
	AssistantAnswer string `json:"assistant_answer"`
	// CreatedAt 是本轮问答完成时的 UTC 时间。
	CreatedAt time.Time `json:"created_at"`
}

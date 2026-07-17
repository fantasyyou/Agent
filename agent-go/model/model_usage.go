package model

import "time"

// ModelUsage 表示持久化在 MySQL 中的单次大模型调用计量记录。
type ModelUsage struct {
	// ID 是该计量记录的全局唯一标识。
	ID string `json:"usage_id"`
	// UserID 是发起本次模型调用的用户唯一标识。
	UserID string `json:"user_id"`
	// SessionID 是触发本次模型调用的会话唯一标识。
	SessionID string `json:"session_id"`
	// RequestID 是 Go、Python 和日志之间关联单次请求的追踪标识。
	RequestID string `json:"request_id"`
	// Provider 是模型供应商，当前可选值为 ModelProviderDeepSeek。
	Provider string `json:"provider"`
	// Model 是供应商实际返回的模型名称，例如 deepseek-chat。
	Model string `json:"model"`
	// InputTokens 是供应商返回的输入 Token 总数。
	InputTokens int64 `json:"input_tokens"`
	// CachedTokens 是 InputTokens 中命中供应商缓存的 Token 数。
	CachedTokens int64 `json:"cached_tokens"`
	// OutputTokens 是模型生成回答所消耗的输出 Token 数。
	OutputTokens int64 `json:"output_tokens"`
	// TotalTokens 是供应商返回的输入和输出 Token 总数。
	TotalTokens int64 `json:"total_tokens"`
	// TotalCost 是根据 Token 单价计算的费用；未配置价格时为 0。
	TotalCost float64 `json:"total_cost"`
	// Currency 是费用币种，当前可选值为 CurrencyCNY。
	Currency string `json:"currency"`
	// LatencyMS 是 Python 调用模型供应商接口的耗时，单位为毫秒。
	LatencyMS int64 `json:"latency_ms"`
	// Status 是模型调用状态，可选值为 UsageStatusSuccess、UsageStatusFailed。
	Status string `json:"status"`
	// CreatedAt 是生成该计量记录时的 UTC 时间。
	CreatedAt time.Time `json:"created_at"`
}

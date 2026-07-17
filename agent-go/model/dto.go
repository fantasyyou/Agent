package model

// Credentials 表示登录或注册接口接收的账号凭证，只能在接口边界短暂使用。
type Credentials struct {
	// Username 是用户提交的登录账号。
	Username string `json:"username"`
	// Password 是用户提交的明文密码，不得持久化或写入日志。
	Password string `json:"password"`
}

// ChatRequest 表示网页向聊天接口提交的请求。
type ChatRequest struct {
	// SessionID 是当前浏览器会话的唯一标识。
	SessionID string `json:"session_id"`
	// Question 是用户本次提交的问题。
	Question string `json:"question"`
}

// ChatResponse 表示聊天接口返回给网页的结果。
type ChatResponse struct {
	// Answer 是请求成功时返回的智能客服回答。
	Answer string `json:"answer,omitempty"`
	// Error 是请求失败时返回的安全提示，不包含内部错误详情。
	Error string `json:"error,omitempty"`
}

// AuthClaims 表示保存在签名登录 Cookie 中的身份声明。
type AuthClaims struct {
	// UserID 是已经通过认证的用户唯一标识。
	UserID string `json:"uid"`
	// Username 是用于网页展示的登录账号。
	Username string `json:"username"`
	// Expires 是登录凭证失效时间对应的 Unix 时间戳。
	Expires int64 `json:"exp"`
}

// AgentRequest 表示 Go 发送给 Python 推理服务的最小必要上下文。
type AgentRequest struct {
	// RequestID 是关联 Go 请求、Python 调用和计量记录的追踪标识。
	RequestID string
	// UserID 是已认证用户标识，不包含完整用户资料。
	UserID string
	// SessionID 是当前客服会话的唯一标识。
	SessionID string
	// Question 是用户当前提交的问题。
	Question string
	// Memories 是 Go 完成鉴权和数量限制后召回的 Top-N 条记忆。
	Memories []ConversationMemory
}

// AgentResponse 表示 Python 推理服务返回的回答和模型计量信息。
type AgentResponse struct {
	// Answer 是模型生成的最终客服回答。
	Answer string
	// Provider 是模型供应商，当前可选值为 ModelProviderDeepSeek。
	Provider string
	// Model 是供应商实际使用的模型名称。
	Model string
	// InputTokens 是供应商返回的输入 Token 总数。
	InputTokens int64
	// CachedTokens 是输入 Token 中命中供应商缓存的数量。
	CachedTokens int64
	// OutputTokens 是模型生成回答使用的输出 Token 数量。
	OutputTokens int64
	// TotalTokens 是供应商返回的输入和输出 Token 总数。
	TotalTokens int64
	// LatencyMS 是模型供应商接口调用耗时，单位为毫秒。
	LatencyMS int64
}

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
	// DialogueState 是 Go 从 Redis 读取的当前多轮任务状态。
	DialogueState *DialogueState
}

const (
	DialogueSchemaVersion    = "v1"
	WorkflowVersionV1        = "v1"
	WorkflowStatusCollecting = "collecting"
	SlotStatusEmpty          = "empty"
	SlotStatusConfirmed      = "confirmed"
	SlotStatusInvalid        = "invalid"
	SlotSourceUser           = "user"
	NextActionAskUser        = "ask_user"
	NextActionRouteWorkflow  = "route_workflow"
	NextActionAnswerUser     = "answer_user"
	NextActionRequestHuman   = "request_human"
)

// WorkflowState 表示当前短期任务的工作流身份和执行状态。
type WorkflowState struct {
	Intent  string `json:"intent"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

// SlotState 表示一个槽位的值、校验状态和可审计来源。
type SlotState struct {
	Value      any     `json:"value"`
	Status     string  `json:"status"`
	Source     string  `json:"source"`
	Evidence   string  `json:"evidence"`
	Confidence float64 `json:"confidence"`
	UpdatedAt  string  `json:"updated_at,omitempty"`
}

// DialogueState 表示 Go 持久化和维护的短期任务工作区，不属于长期用户记忆。
type DialogueState struct {
	SchemaVersion string               `json:"schema_version"`
	Workflow      WorkflowState        `json:"workflow"`
	ActiveSlot    string               `json:"active_slot"`
	Slots         map[string]SlotState `json:"slots"`
	RetryCount    map[string]int       `json:"retry_count"`
	LastAction    string               `json:"last_action"`
	LastQuestion  string               `json:"last_question"`
	UpdatedAt     string               `json:"updated_at,omitempty"`
}

// DialogueDecision 是 Python 返回的理解结果和下一步建议，最终状态由 Go 合并。
type DialogueDecision struct {
	Intent           string
	Status           string
	ActiveSlot       string
	SlotUpdates      map[string]SlotState
	NextAction       string
	NextQuestion     string
	SuggestedOptions []string
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
	// Decision 是 Python 对本轮输入的理解和建议动作，不直接代表持久化状态。
	Decision DialogueDecision
}

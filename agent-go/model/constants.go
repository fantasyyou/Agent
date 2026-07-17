package model

const (
	// UserStatusActive 表示用户状态正常，允许登录和使用客服服务。
	UserStatusActive = "active"
	// UserStatusDisabled 表示用户已被禁用，不允许登录。
	UserStatusDisabled = "disabled"

	// SessionStatusActive 表示会话仍在使用中。
	SessionStatusActive = "active"
	// SessionStatusClosed 表示会话已经结束，不应继续追加消息。
	SessionStatusClosed = "closed"

	// UsageStatusSuccess 表示模型调用成功并且取得了回答。
	UsageStatusSuccess = "success"
	// UsageStatusFailed 表示模型调用失败，没有取得有效回答。
	UsageStatusFailed = "failed"

	// ModelProviderDeepSeek 表示模型供应商为 DeepSeek。
	ModelProviderDeepSeek = "deepseek"

	// CurrencyCNY 表示费用币种为人民币。
	CurrencyCNY = "CNY"
)

// Package model 定义各层共享的业务模型、接口数据模型和枚举常量。
package model

import "time"

// User 表示持久化在 MySQL 中的注册用户。
type User struct {
	// ID 是服务端生成的用户全局唯一标识。
	ID string `json:"user_id"`
	// Username 是注册和登录使用的唯一账号。
	Username string `json:"username"`
	// PasswordHash 是 bcrypt 密码哈希；系统不保存明文密码。
	PasswordHash string `json:"-"`
	// Status 是用户状态，可选值为 UserStatusActive、UserStatusDisabled。
	Status string `json:"status"`
	// CreatedAt 是用户注册时的 UTC 时间。
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 是用户信息最后一次更新时的 UTC 时间。
	UpdatedAt time.Time `json:"updated_at"`
}

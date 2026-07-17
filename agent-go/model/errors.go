package model

import "errors"

var (
	// ErrUsernameExists 表示注册账号违反了用户名唯一约束。
	ErrUsernameExists = errors.New("username already exists")
	// ErrUserNotFound 表示用户不存在；登录时也用它统一表示凭证无效，避免泄露账号是否存在。
	ErrUserNotFound = errors.New("user not found")
)

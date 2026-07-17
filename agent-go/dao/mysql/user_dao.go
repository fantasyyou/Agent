package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"agent-go/model"
	gomysql "github.com/go-sql-driver/mysql"
)

type UserDAO struct{ db *sql.DB }

func NewUserDAO(db *sql.DB) *UserDAO { return &UserDAO{db: db} }

func (d *UserDAO) Create(ctx context.Context, user model.User) error {
	_, err := d.db.ExecContext(ctx, `INSERT INTO users(id,username,password_hash,status,created_at,updated_at) VALUES(?,?,?,?,?,?)`, user.ID, user.Username, user.PasswordHash, user.Status, user.CreatedAt, user.UpdatedAt)
	if driverErr := new(gomysql.MySQLError); errors.As(err, &driverErr) && driverErr.Number == 1062 {
		return model.ErrUsernameExists
	}
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
func (d *UserDAO) GetByUsername(ctx context.Context, username string) (model.User, error) {
	var user model.User
	err := d.db.QueryRowContext(ctx, `SELECT id,username,password_hash,status,created_at,updated_at FROM users WHERE username=?`, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.ErrUserNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

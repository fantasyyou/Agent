// Package mysql contains MySQL persistence implementations owned by the Go service.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"agent-go/config"
	_ "github.com/go-sql-driver/mysql"
)

func Open(ctx context.Context, cfg config.MySQLConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open MySQL: %w", err)
	}
	db.SetMaxOpenConns(cfg.MaxOpenConnections)
	db.SetMaxIdleConns(cfg.MaxIdleConnections)
	db.SetConnMaxLifetime(config.Duration(cfg.ConnectionMaxLifetime, 5*time.Minute))
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping MySQL: %w", err)
	}
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(40) PRIMARY KEY, username VARCHAR(32) NOT NULL UNIQUE,
			password_hash VARCHAR(100) NOT NULL, status VARCHAR(16) NOT NULL,
			created_at DATETIME(6) NOT NULL, updated_at DATETIME(6) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS conversation_sessions (
			id VARCHAR(80) NOT NULL, user_id VARCHAR(40) NOT NULL, status VARCHAR(16) NOT NULL,
			created_at DATETIME(6) NOT NULL, last_active_at DATETIME(6) NOT NULL,
			PRIMARY KEY (user_id, id), INDEX idx_session_last_active (user_id, last_active_at),
			CONSTRAINT fk_session_user FOREIGN KEY (user_id) REFERENCES users(id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS model_usages (
			id VARCHAR(40) PRIMARY KEY, user_id VARCHAR(40) NOT NULL, session_id VARCHAR(80) NOT NULL,
			request_id VARCHAR(40) NOT NULL UNIQUE, provider VARCHAR(32) NOT NULL, model VARCHAR(64) NOT NULL,
			input_tokens BIGINT NOT NULL DEFAULT 0, cached_tokens BIGINT NOT NULL DEFAULT 0,
			output_tokens BIGINT NOT NULL DEFAULT 0, total_tokens BIGINT NOT NULL DEFAULT 0,
			total_cost DECIMAL(18,8) NOT NULL DEFAULT 0, currency CHAR(3) NOT NULL DEFAULT 'CNY',
			latency_ms BIGINT NOT NULL DEFAULT 0, status VARCHAR(16) NOT NULL, created_at DATETIME(6) NOT NULL,
			INDEX idx_usage_user_created (user_id, created_at),
			CONSTRAINT fk_usage_user FOREIGN KEY (user_id) REFERENCES users(id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS action_executions (
			id VARCHAR(40) PRIMARY KEY, user_id VARCHAR(40) NOT NULL, session_id VARCHAR(80) NOT NULL,
			request_id VARCHAR(40) NOT NULL, intent VARCHAR(64) NOT NULL, action VARCHAR(32) NOT NULL,
			active_slot VARCHAR(64) NOT NULL DEFAULT '', status VARCHAR(32) NOT NULL,
			result_message VARCHAR(255) NOT NULL DEFAULT '', created_at DATETIME(6) NOT NULL,
			INDEX idx_action_request (request_id), INDEX idx_action_user_created (user_id, created_at),
			CONSTRAINT fk_action_user FOREIGN KEY (user_id) REFERENCES users(id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate MySQL: %w", err)
		}
	}
	return nil
}

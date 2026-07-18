// Package redis provides short-lived dialogue state persistence.
package redis

import (
	"agent-go/config"
	"agent-go/model"
	"context"
	"encoding/json"
	"fmt"
	"time"

	redislib "github.com/redis/go-redis/v9"
)

type DialogueStateDAO struct {
	client  *redislib.Client
	ttl     time.Duration
	timeout time.Duration
}

func NewDialogueStateDAO(cfg config.RedisConfig) *DialogueStateDAO {
	client := redislib.NewClient(&redislib.Options{
		Addr: cfg.Address, Password: cfg.Password, DB: cfg.DB,
		DialTimeout:  config.Duration(cfg.Timeout, 3*time.Second),
		ReadTimeout:  config.Duration(cfg.Timeout, 3*time.Second),
		WriteTimeout: config.Duration(cfg.Timeout, 3*time.Second),
	})
	return &DialogueStateDAO{client: client, ttl: config.Duration(cfg.StateTTL, 30*time.Minute), timeout: config.Duration(cfg.Timeout, 3*time.Second)}
}

func (d *DialogueStateDAO) Ping(ctx context.Context) error {
	callCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()
	if err := d.client.Ping(callCtx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

func (d *DialogueStateDAO) Close() error { return d.client.Close() }

func (d *DialogueStateDAO) Get(ctx context.Context, userID, sessionID string) (*model.DialogueState, error) {
	data, err := d.client.Get(ctx, stateKey(userID, sessionID)).Bytes()
	if err == redislib.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get dialogue state: %w", err)
	}
	var state model.DialogueState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode dialogue state: %w", err)
	}
	return &state, nil
}

func (d *DialogueStateDAO) Set(ctx context.Context, userID, sessionID string, state model.DialogueState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode dialogue state: %w", err)
	}
	if err := d.client.Set(ctx, stateKey(userID, sessionID), data, d.ttl).Err(); err != nil {
		return fmt.Errorf("set dialogue state: %w", err)
	}
	return nil
}

func (d *DialogueStateDAO) Delete(ctx context.Context, userID, sessionID string) error {
	if err := d.client.Del(ctx, stateKey(userID, sessionID)).Err(); err != nil {
		return fmt.Errorf("delete dialogue state: %w", err)
	}
	return nil
}

func stateKey(userID, sessionID string) string {
	return "agent:dialogue:" + userID + ":" + sessionID
}

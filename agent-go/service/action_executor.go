package service

import (
	"agent-go/model"
	"context"
	"fmt"
	"log/slog"
	"time"
)

type ActionExecutionRepository interface {
	Create(context.Context, model.ActionExecution) error
}

// ActionExecutor 是 Go 侧业务动作执行入口；真实产品库或工单接口后续在此接入。
type ActionExecutor struct{ repository ActionExecutionRepository }

func NewActionExecutor(repository ActionExecutionRepository) *ActionExecutor {
	return &ActionExecutor{repository: repository}
}

func (e *ActionExecutor) Execute(ctx context.Context, userID, sessionID, requestID string, decision model.DialogueDecision) error {
	status := model.ActionExecutionStatusSuccess
	message := "dialogue action completed"
	if decision.NextAction == model.NextActionRouteWorkflow || decision.NextAction == model.NextActionRequestHuman {
		status = model.ActionExecutionStatusNotConfigured
		message = "external business adapter is not configured"
	}
	id, err := newID("act_")
	if err != nil {
		return err
	}
	execution := model.ActionExecution{ID: id, UserID: userID, SessionID: sessionID, RequestID: requestID, Intent: decision.Intent, Action: decision.NextAction, ActiveSlot: decision.ActiveSlot, Status: status, ResultMessage: message, CreatedAt: time.Now().UTC()}
	if err := e.repository.Create(ctx, execution); err != nil {
		return fmt.Errorf("store action execution: %w", err)
	}
	slog.Info("dialogue_action_executed", "request_id", requestID, "intent", decision.Intent, "action", decision.NextAction, "active_slot", decision.ActiveSlot, "status", status)
	return nil
}

package service

import (
	"agent-go/model"
	"agent-go/observability"
	"context"
	"fmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"time"
)

type MemoryRepository interface {
	Search(context.Context, string, string, string, int) ([]model.ConversationMemory, error)
	Store(context.Context, model.ConversationMemory) error
}
type SessionRepository interface {
	Touch(context.Context, model.ConversationSession) error
}
type UsageRepository interface {
	Create(context.Context, model.ModelUsage) error
}
type AgentClient interface {
	Answer(context.Context, model.AgentRequest) (model.AgentResponse, error)
}
type DialogueStateRepository interface {
	Get(context.Context, string, string) (*model.DialogueState, error)
	Set(context.Context, string, string, model.DialogueState) error
	Delete(context.Context, string, string) error
}
type ChatService struct {
	memory       MemoryRepository
	sessions     SessionRepository
	usage        UsageRepository
	agent        AgentClient
	limit        int
	states       DialogueStateRepository
	stateService *DialogueStateService
	actions      *ActionExecutor
}

func NewChatService(memory MemoryRepository, sessions SessionRepository, usage UsageRepository, states DialogueStateRepository, agent AgentClient, actions *ActionExecutor, limit int) *ChatService {
	return &ChatService{memory: memory, sessions: sessions, usage: usage, states: states, stateService: NewDialogueStateService(), agent: agent, actions: actions, limit: limit}
}
func (s *ChatService) Ask(ctx context.Context, userID, sessionID, question string) (answer string, resultErr error) {
	ctx = observability.WithRequest(ctx)
	ctx, finish := observability.Stage(ctx, "chat.process")
	defer func() {
		if resultErr != nil {
			trace.SpanFromContext(ctx).SetAttributes(attribute.String("request.outcome", "failed"))
		}
		finish(resultErr)
	}()
	started := time.Now()
	requestID := observability.RequestID(ctx)
	slog.InfoContext(ctx, "chat_processing_started", "user_id", userID, "session_id", sessionID, "question_chars", len([]rune(question)))
	now := time.Now().UTC()
	stageCtx, done := observability.Stage(ctx, "session.touch")
	err := s.sessions.Touch(stageCtx, model.ConversationSession{ID: sessionID, UserID: userID, Status: model.SessionStatusActive, CreatedAt: now, LastActiveAt: now})
	done(err)
	if err != nil {
		slog.ErrorContext(ctx, "chat_stage_failed", "stage", "session_touch", "error", err)
		return "", err
	}
	stageCtx, done = observability.Stage(ctx, "memory.search")
	memories, err := s.memory.Search(stageCtx, userID, sessionID, question, s.limit)
	done(err)
	if err != nil {
		slog.ErrorContext(ctx, "chat_stage_failed", "stage", "memory_search", "error", err)
		return "", fmt.Errorf("search memory: %w", err)
	}
	slog.InfoContext(ctx, "memory_recalled", "memory_count", len(memories))
	stageCtx, done = observability.Stage(ctx, "dialogue_state.get")
	state, err := s.states.Get(stageCtx, userID, sessionID)
	done(err)
	if err != nil {
		return "", fmt.Errorf("get dialogue state: %w", err)
	}
	slog.InfoContext(ctx, "dialogue_state_loaded", "has_state", state != nil)
	stageCtx, done = observability.Stage(ctx, "python.answer")
	response, err := s.agent.Answer(stageCtx, model.AgentRequest{RequestID: requestID, UserID: userID, SessionID: sessionID, Question: question, Memories: memories, DialogueState: state})
	done(err)
	if err != nil {
		slog.ErrorContext(ctx, "chat_stage_failed", "stage", "python_agent", "error", err)
		return "", fmt.Errorf("call Python agent: %w", err)
	}
	stageCtx, done = observability.Stage(ctx, "dialogue_state.apply")
	nextState, completed := s.stateService.Apply(state, response.Decision, time.Now().UTC())
	done(nil)
	if completed {
		stageCtx, done = observability.Stage(ctx, "dialogue_state.delete")
		err = s.states.Delete(stageCtx, userID, sessionID)
		done(err)
		if err != nil {
			return "", fmt.Errorf("delete dialogue state: %w", err)
		}
		slog.InfoContext(ctx, "dialogue_state_deleted", "request_id", requestID)
	} else {
		stageCtx, done = observability.Stage(ctx, "dialogue_state.set")
		err = s.states.Set(stageCtx, userID, sessionID, nextState)
		done(err)
		if err != nil {
			return "", fmt.Errorf("store dialogue state: %w", err)
		}
		slog.InfoContext(ctx, "dialogue_state_stored", "intent", nextState.Workflow.Intent, "active_slot", nextState.ActiveSlot, "slot_count", len(nextState.Slots), "retry_count", nextState.RetryCount[nextState.ActiveSlot])
	}
	stageCtx, done = observability.Stage(ctx, "action.execute")
	err = s.actions.Execute(stageCtx, userID, sessionID, requestID, response.Decision)
	done(err)
	if err != nil {
		slog.ErrorContext(ctx, "chat_stage_failed", "stage", "action_execute", "error", err)
		return "", err
	}
	stageCtx, done = observability.Stage(ctx, "memory.store")
	err = s.memory.Store(stageCtx, model.ConversationMemory{UserID: userID, SessionID: sessionID, UserQuestion: question, AssistantAnswer: response.Answer, CreatedAt: time.Now().UTC()})
	done(err)
	if err != nil {
		slog.ErrorContext(ctx, "chat_stage_failed", "stage", "memory_store", "error", err)
		return "", fmt.Errorf("store memory: %w", err)
	}
	outcome := "answered"
	if response.Decision.NextAction == model.NextActionAskUser {
		outcome = "awaiting_input"
	}
	if response.Decision.NextAction == model.NextActionRouteWorkflow || response.Decision.NextAction == model.NextActionRequestHuman {
		outcome = "business_not_configured"
	}
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("business.outcome", outcome), attribute.String("session_id", sessionID))
	slog.InfoContext(ctx, "business_outcome", "outcome", outcome, "session_id", sessionID)
	usageID, err := newID("use_")
	if err != nil {
		return "", err
	}
	usage := model.ModelUsage{ID: usageID, UserID: userID, SessionID: sessionID, RequestID: requestID, Provider: response.Provider, Model: response.Model, InputTokens: response.InputTokens, CachedTokens: response.CachedTokens, OutputTokens: response.OutputTokens, TotalTokens: response.TotalTokens, TotalCost: 0, Currency: model.CurrencyCNY, LatencyMS: response.LatencyMS, Status: model.UsageStatusSuccess, CreatedAt: time.Now().UTC()}
	stageCtx, done = observability.Stage(ctx, "usage.store")
	err = s.usage.Create(stageCtx, usage)
	done(err)
	if err != nil {
		slog.ErrorContext(ctx, "chat_stage_failed", "stage", "usage_store", "error", err)
		return "", fmt.Errorf("store model usage: %w", err)
	}
	slog.InfoContext(ctx, "chat_processing_completed", "user_id", userID, "session_id", sessionID, "model", response.Model, "input_tokens", response.InputTokens, "cached_tokens", response.CachedTokens, "output_tokens", response.OutputTokens, "total_tokens", response.TotalTokens, "llm_latency_ms", response.LatencyMS, "duration_ms", time.Since(started).Milliseconds())
	return response.Answer, nil
}

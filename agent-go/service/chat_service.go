package service

import (
	"agent-go/model"
	"context"
	"fmt"
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
func (s *ChatService) Ask(ctx context.Context, userID, sessionID, question string) (string, error) {
	started := time.Now()
	requestID, err := newID("req_")
	if err != nil {
		return "", err
	}
	slog.Info("chat_processing_started", "request_id", requestID, "user_id", userID, "session_id", sessionID, "question_chars", len([]rune(question)))
	now := time.Now().UTC()
	if err := s.sessions.Touch(ctx, model.ConversationSession{ID: sessionID, UserID: userID, Status: model.SessionStatusActive, CreatedAt: now, LastActiveAt: now}); err != nil {
		slog.Error("chat_stage_failed", "request_id", requestID, "stage", "session_touch", "error", err)
		return "", err
	}
	memories, err := s.memory.Search(ctx, userID, sessionID, question, s.limit)
	if err != nil {
		slog.Error("chat_stage_failed", "request_id", requestID, "stage", "memory_search", "error", err)
		return "", fmt.Errorf("search memory: %w", err)
	}
	slog.Info("memory_recalled", "request_id", requestID, "memory_count", len(memories))
	state, err := s.states.Get(ctx, userID, sessionID)
	if err != nil {
		return "", fmt.Errorf("get dialogue state: %w", err)
	}
	slog.Info("dialogue_state_loaded", "request_id", requestID, "has_state", state != nil)
	response, err := s.agent.Answer(ctx, model.AgentRequest{RequestID: requestID, UserID: userID, SessionID: sessionID, Question: question, Memories: memories, DialogueState: state})
	if err != nil {
		slog.Error("chat_stage_failed", "request_id", requestID, "stage", "python_agent", "error", err)
		return "", fmt.Errorf("call Python agent: %w", err)
	}
	nextState, completed := s.stateService.Apply(state, response.Decision, time.Now().UTC())
	if completed {
		if err := s.states.Delete(ctx, userID, sessionID); err != nil {
			return "", fmt.Errorf("delete dialogue state: %w", err)
		}
		slog.Info("dialogue_state_deleted", "request_id", requestID)
	} else {
		if err := s.states.Set(ctx, userID, sessionID, nextState); err != nil {
			return "", fmt.Errorf("store dialogue state: %w", err)
		}
		slog.Info("dialogue_state_stored", "request_id", requestID, "intent", nextState.Workflow.Intent, "active_slot", nextState.ActiveSlot, "slot_count", len(nextState.Slots), "retry_count", nextState.RetryCount[nextState.ActiveSlot])
	}
	if err := s.actions.Execute(ctx, userID, sessionID, requestID, response.Decision); err != nil {
		slog.Error("chat_stage_failed", "request_id", requestID, "stage", "action_execute", "error", err)
		return "", err
	}
	if err := s.memory.Store(ctx, model.ConversationMemory{UserID: userID, SessionID: sessionID, UserQuestion: question, AssistantAnswer: response.Answer, CreatedAt: time.Now().UTC()}); err != nil {
		slog.Error("chat_stage_failed", "request_id", requestID, "stage", "memory_store", "error", err)
		return "", fmt.Errorf("store memory: %w", err)
	}
	usageID, err := newID("use_")
	if err != nil {
		return "", err
	}
	usage := model.ModelUsage{ID: usageID, UserID: userID, SessionID: sessionID, RequestID: requestID, Provider: response.Provider, Model: response.Model, InputTokens: response.InputTokens, CachedTokens: response.CachedTokens, OutputTokens: response.OutputTokens, TotalTokens: response.TotalTokens, TotalCost: 0, Currency: model.CurrencyCNY, LatencyMS: response.LatencyMS, Status: model.UsageStatusSuccess, CreatedAt: time.Now().UTC()}
	if err := s.usage.Create(ctx, usage); err != nil {
		slog.Error("chat_stage_failed", "request_id", requestID, "stage", "usage_store", "error", err)
		return "", fmt.Errorf("store model usage: %w", err)
	}
	slog.Info("chat_processing_completed", "request_id", requestID, "user_id", userID, "session_id", sessionID, "model", response.Model, "input_tokens", response.InputTokens, "cached_tokens", response.CachedTokens, "output_tokens", response.OutputTokens, "total_tokens", response.TotalTokens, "llm_latency_ms", response.LatencyMS, "duration_ms", time.Since(started).Milliseconds())
	return response.Answer, nil
}

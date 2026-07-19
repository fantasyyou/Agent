// Package client contains outbound clients for other services.
package client

import (
	"agent-go/config"
	"agent-go/model"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"time"
)

const answerMethod = "/customer.CustomerService/Answer"

type PythonAgent struct {
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewPythonAgent(cfg config.PythonAgentConfig) (*PythonAgent, error) {
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create gRPC client: %w", err)
	}
	return &PythonAgent{conn: conn, timeout: config.Duration(cfg.Timeout, 60*time.Second)}, nil
}
func (c *PythonAgent) Close() error { return c.conn.Close() }
func (c *PythonAgent) Answer(ctx context.Context, input model.AgentRequest) (model.AgentResponse, error) {
	memories := make([]any, 0, len(input.Memories))
	for _, memory := range input.Memories {
		memories = append(memories, map[string]any{"user_question": memory.UserQuestion, "assistant_answer": memory.AssistantAnswer, "created_at": memory.CreatedAt.Format(time.RFC3339)})
	}
	dialogueState := map[string]any{}
	if input.DialogueState != nil {
		dialogueState = dialogueStateMap(*input.DialogueState)
	}
	req, err := structpb.NewStruct(map[string]any{"request_id": input.RequestID, "user_id": input.UserID, "session_id": input.SessionID, "question": input.Question, "memories": memories, "dialogue_state": dialogueState})
	if err != nil {
		return model.AgentResponse{}, err
	}
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	response := &structpb.Struct{}
	if err := c.conn.Invoke(callCtx, answerMethod, req, response); err != nil {
		return model.AgentResponse{}, err
	}
	values := response.AsMap()
	answer, _ := values["answer"].(string)
	if answer == "" {
		return model.AgentResponse{}, fmt.Errorf("Python agent returned an empty answer")
	}
	decision := dialogueDecision(mapValue(values, "decision"))
	return model.AgentResponse{Answer: answer, Provider: stringValue(values, "provider"), Model: stringValue(values, "model"), InputTokens: intValue(values, "input_tokens"), CachedTokens: intValue(values, "cached_tokens"), OutputTokens: intValue(values, "output_tokens"), TotalTokens: intValue(values, "total_tokens"), LatencyMS: intValue(values, "latency_ms"), Decision: decision}, nil
}
func stringMapValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}
func mapValue(values map[string]any, key string) map[string]any {
	value, _ := values[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func dialogueStateMap(state model.DialogueState) map[string]any {
	slots := make(map[string]any, len(state.Slots))
	for name, slot := range state.Slots {
		slots[name] = map[string]any{"value": slot.Value, "status": slot.Status, "source": slot.Source, "evidence": slot.Evidence, "confidence": slot.Confidence, "updated_at": slot.UpdatedAt}
	}
	retryCount := make(map[string]any, len(state.RetryCount))
	for name, count := range state.RetryCount {
		retryCount[name] = count
	}
	return map[string]any{
		"schema_version": state.SchemaVersion,
		"workflow":       map[string]any{"intent": state.Workflow.Intent, "version": state.Workflow.Version, "status": state.Workflow.Status},
		"active_slot":    state.ActiveSlot, "slots": slots, "retry_count": retryCount,
		"last_action": state.LastAction, "last_question": state.LastQuestion, "updated_at": state.UpdatedAt,
	}
}

func dialogueDecision(raw map[string]any) model.DialogueDecision {
	updates := map[string]model.SlotState{}
	for name, value := range mapValue(raw, "slot_updates") {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		updates[name] = model.SlotState{Value: item["value"], Status: stringMapValue(item, "status"), Source: stringMapValue(item, "source"), Evidence: stringMapValue(item, "evidence"), Confidence: floatMapValue(item, "confidence")}
	}
	options := []string{}
	if rawOptions, ok := raw["suggested_options"].([]any); ok {
		for _, option := range rawOptions {
			if text, ok := option.(string); ok {
				options = append(options, text)
			}
		}
	}
	return model.DialogueDecision{Intent: stringMapValue(raw, "intent"), Status: stringMapValue(raw, "status"), ActiveSlot: stringMapValue(raw, "active_slot"), SlotUpdates: updates, NextAction: stringMapValue(raw, "next_action"), NextQuestion: stringMapValue(raw, "next_question"), SuggestedOptions: options}
}

func floatMapValue(values map[string]any, key string) float64 {
	value, _ := values[key].(float64)
	return value
}
func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}
func intValue(values map[string]any, key string) int64 {
	value, _ := values[key].(float64)
	return int64(value)
}

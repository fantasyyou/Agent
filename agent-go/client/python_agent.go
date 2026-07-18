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
		dialogueState = map[string]any{"intent": input.DialogueState.Intent, "slots": input.DialogueState.Slots, "status": input.DialogueState.Status, "last_question": input.DialogueState.LastQuestion}
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
	var state *model.DialogueState
	if raw, ok := values["dialogue_state"].(map[string]any); ok && len(raw) > 0 {
		state = &model.DialogueState{Intent: stringMapValue(raw, "intent"), Slots: mapValue(raw, "slots"), Status: stringMapValue(raw, "status"), LastQuestion: stringMapValue(raw, "last_question")}
	}
	clearState, _ := values["clear_dialogue_state"].(bool)
	return model.AgentResponse{Answer: answer, Provider: stringValue(values, "provider"), Model: stringValue(values, "model"), InputTokens: intValue(values, "input_tokens"), CachedTokens: intValue(values, "cached_tokens"), OutputTokens: intValue(values, "output_tokens"), TotalTokens: intValue(values, "total_tokens"), LatencyMS: intValue(values, "latency_ms"), DialogueState: state, ClearDialogueState: clearState}, nil
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
func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}
func intValue(values map[string]any, key string) int64 {
	value, _ := values[key].(float64)
	return int64(value)
}

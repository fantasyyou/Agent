package main

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

const answerMethod = "/customer.CustomerService/Answer"

type PythonAgentClient struct {
	conn    *grpc.ClientConn
	timeout time.Duration
}

func NewPythonAgentClient(cfg PythonAgentConfig) (*PythonAgentClient, error) {
	timeout, _ := parseDuration(cfg.Timeout, 60*time.Second)
	conn, err := grpc.NewClient(cfg.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create gRPC client: %w", err)
	}
	return &PythonAgentClient{conn: conn, timeout: timeout}, nil
}

func (c *PythonAgentClient) Close() error { return c.conn.Close() }

func (c *PythonAgentClient) Answer(ctx context.Context, sessionID, question string, memories []Memory) (string, error) {
	memoryValues := make([]any, 0, len(memories))
	for _, memory := range memories {
		memoryValues = append(memoryValues, map[string]any{
			"user_question": memory.UserQuestion, "assistant_answer": memory.AssistantAnswer,
			"created_at": memory.CreatedAt.Format(time.RFC3339),
		})
	}
	req, err := structpb.NewStruct(map[string]any{
		"session_id": sessionID, "question": question, "memories": memoryValues,
	})
	if err != nil {
		return "", err
	}
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	resp := &structpb.Struct{}
	if err := c.conn.Invoke(callCtx, answerMethod, req, resp); err != nil {
		return "", err
	}
	answer, ok := resp.AsMap()["answer"].(string)
	if !ok || answer == "" {
		return "", fmt.Errorf("Python agent returned an empty answer")
	}
	return answer, nil
}

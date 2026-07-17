package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type CustomerService struct {
	memory *ESMemory
	agent  *PythonAgentClient
	limit  int
}

func (s *CustomerService) Ask(ctx context.Context, sessionID, question string) (string, error) {
	memories, err := s.memory.Search(ctx, sessionID, question, s.limit)
	if err != nil {
		return "", fmt.Errorf("search memory: %w", err)
	}

	answer, err := s.agent.Answer(ctx, sessionID, question, memories)
	if err != nil {
		return "", fmt.Errorf("call Python agent: %w", err)
	}

	if err := s.memory.Store(ctx, Memory{
		SessionID:       sessionID,
		UserQuestion:    question,
		AssistantAnswer: answer,
		CreatedAt:       time.Now().UTC(),
	}); err != nil {
		return "", fmt.Errorf("store memory: %w", err)
	}
	return answer, nil
}

func main() {
	configPath := flag.String("config", "config.json", "configuration file")
	sessionID := flag.String("session", "demo-user", "conversation session ID")
	question := flag.String("question", "", "single question; omit for interactive mode")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		exitWithError(err)
	}

	ctx := context.Background()
	memory := NewESMemory(cfg.Elasticsearch)
	if err := memory.EnsureIndex(ctx); err != nil {
		exitWithError(fmt.Errorf("initialize Elasticsearch memory index: %w", err))
	}
	agent, err := NewPythonAgentClient(cfg.PythonAgent)
	if err != nil {
		exitWithError(err)
	}
	defer agent.Close()

	service := &CustomerService{memory: memory, agent: agent, limit: cfg.Memory.RecallLimit}
	if strings.TrimSpace(*question) != "" {
		answer, err := service.Ask(ctx, *sessionID, *question)
		if err != nil {
			exitWithError(err)
		}
		fmt.Println(answer)
		return
	}

	fmt.Printf("客服已启动（session=%s，输入 exit 退出）\n", *sessionID)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		if strings.EqualFold(text, "exit") || strings.EqualFold(text, "quit") {
			break
		}
		answer, err := service.Ask(ctx, *sessionID, text)
		if err != nil {
			fmt.Fprintln(os.Stderr, "错误:", err)
			continue
		}
		fmt.Println(answer)
	}
	if err := scanner.Err(); err != nil {
		exitWithError(err)
	}
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}

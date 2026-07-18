package service

import (
	"agent-go/model"
	"context"
	"testing"
)

type memoryStub struct{}

func (memoryStub) Search(context.Context, string, string, string, int) ([]model.ConversationMemory, error) {
	return nil, nil
}
func (memoryStub) Store(context.Context, model.ConversationMemory) error { return nil }

type sessionStub struct{}

func (sessionStub) Touch(context.Context, model.ConversationSession) error { return nil }

type usageStub struct{}

func (usageStub) Create(context.Context, model.ModelUsage) error { return nil }

type stateStub struct {
	current *model.DialogueState
	set     *model.DialogueState
	deleted bool
}

func (s *stateStub) Get(context.Context, string, string) (*model.DialogueState, error) {
	return s.current, nil
}
func (s *stateStub) Set(_ context.Context, _, _ string, state model.DialogueState) error {
	s.set = &state
	return nil
}
func (s *stateStub) Delete(context.Context, string, string) error {
	s.deleted = true
	return nil
}

type agentStub struct {
	request  model.AgentRequest
	response model.AgentResponse
}

func (a *agentStub) Answer(_ context.Context, request model.AgentRequest) (model.AgentResponse, error) {
	a.request = request
	return a.response, nil
}

func TestChatServicePassesAndStoresDialogueState(t *testing.T) {
	initial := &model.DialogueState{Intent: "product_recommendation", Slots: map[string]any{"risk_tolerance": "conservative"}}
	next := &model.DialogueState{Intent: "product_recommendation", Slots: map[string]any{"risk_tolerance": "conservative", "investment_period_months": float64(6)}}
	states := &stateStub{current: initial}
	agent := &agentStub{response: model.AgentResponse{Answer: "请问金额是多少？", Provider: "deepseek", Model: "deepseek-chat", DialogueState: next}}
	service := NewChatService(memoryStub{}, sessionStub{}, usageStub{}, states, agent, 5)

	if _, err := service.Ask(context.Background(), "usr_1", "session_1", "半年"); err != nil {
		t.Fatal(err)
	}
	if agent.request.DialogueState == nil || agent.request.DialogueState.Intent != initial.Intent {
		t.Fatalf("initial state was not passed to Python: %#v", agent.request.DialogueState)
	}
	if states.set == nil || states.set.Slots["investment_period_months"] != float64(6) {
		t.Fatalf("next state was not stored: %#v", states.set)
	}
}

func TestChatServiceDeletesCompletedDialogueState(t *testing.T) {
	states := &stateStub{current: &model.DialogueState{Intent: "fee_query"}}
	agent := &agentStub{response: model.AgentResponse{Answer: "已完成", Provider: "deepseek", Model: "deepseek-chat", ClearDialogueState: true}}
	service := NewChatService(memoryStub{}, sessionStub{}, usageStub{}, states, agent, 5)

	if _, err := service.Ask(context.Background(), "usr_1", "session_1", "跨行转账"); err != nil {
		t.Fatal(err)
	}
	if !states.deleted {
		t.Fatal("completed dialogue state was not deleted")
	}
}

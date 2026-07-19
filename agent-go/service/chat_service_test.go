package service

import (
	"agent-go/model"
	"context"
	"testing"
	"time"
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

type actionExecutionStub struct{ created []model.ActionExecution }

func (s *actionExecutionStub) Create(_ context.Context, execution model.ActionExecution) error {
	s.created = append(s.created, execution)
	return nil
}

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
	initial := &model.DialogueState{SchemaVersion: model.DialogueSchemaVersion, Workflow: model.WorkflowState{Intent: "product_recommendation", Version: "v1", Status: "collecting"}, ActiveSlot: "investment_period_months", Slots: map[string]model.SlotState{"risk_tolerance": {Value: "conservative", Status: model.SlotStatusConfirmed}}, RetryCount: map[string]int{}}
	states := &stateStub{current: initial}
	agent := &agentStub{response: model.AgentResponse{Answer: "请问金额是多少？", Provider: "deepseek", Model: "deepseek-chat", Decision: model.DialogueDecision{Intent: "product_recommendation", Status: "need_clarification", ActiveSlot: "investment_amount", SlotUpdates: map[string]model.SlotState{"investment_period_months": {Value: float64(6), Status: model.SlotStatusConfirmed, Source: model.SlotSourceUser, Evidence: "半年"}}, NextAction: model.NextActionAskUser, NextQuestion: "请问金额是多少？"}}}
	actions := &actionExecutionStub{}
	service := NewChatService(memoryStub{}, sessionStub{}, usageStub{}, states, agent, NewActionExecutor(actions), 5)

	if _, err := service.Ask(context.Background(), "usr_1", "session_1", "半年"); err != nil {
		t.Fatal(err)
	}
	if agent.request.DialogueState == nil || agent.request.DialogueState.Workflow.Intent != initial.Workflow.Intent {
		t.Fatalf("initial state was not passed to Python: %#v", agent.request.DialogueState)
	}
	if states.set == nil || states.set.Slots["investment_period_months"].Value != float64(6) || states.set.ActiveSlot != "investment_amount" {
		t.Fatalf("next state was not stored: %#v", states.set)
	}
	if len(actions.created) != 1 || actions.created[0].Action != model.NextActionAskUser {
		t.Fatalf("action was not recorded: %#v", actions.created)
	}
}

func TestChatServiceDeletesCompletedDialogueState(t *testing.T) {
	states := &stateStub{current: &model.DialogueState{SchemaVersion: "v1", Workflow: model.WorkflowState{Intent: "fee_query"}}}
	agent := &agentStub{response: model.AgentResponse{Answer: "已完成", Provider: "deepseek", Model: "deepseek-chat", Decision: model.DialogueDecision{Intent: "fee_query", NextAction: model.NextActionRouteWorkflow}}}
	service := NewChatService(memoryStub{}, sessionStub{}, usageStub{}, states, agent, NewActionExecutor(&actionExecutionStub{}), 5)

	if _, err := service.Ask(context.Background(), "usr_1", "session_1", "跨行转账"); err != nil {
		t.Fatal(err)
	}
	if !states.deleted {
		t.Fatal("completed dialogue state was not deleted")
	}
}

func TestDialogueStateServiceIncrementsRetryForUnansweredActiveSlot(t *testing.T) {
	current := &model.DialogueState{SchemaVersion: "v1", Workflow: model.WorkflowState{Intent: "product_recommendation"}, ActiveSlot: "investment_amount", Slots: map[string]model.SlotState{}, RetryCount: map[string]int{"investment_amount": 1}}
	decision := model.DialogueDecision{Intent: "product_recommendation", ActiveSlot: "investment_amount", SlotUpdates: map[string]model.SlotState{}, NextAction: model.NextActionAskUser, NextQuestion: "请输入金额"}
	next, completed := NewDialogueStateService().Apply(current, decision, time.Now().UTC())
	if completed || next.RetryCount["investment_amount"] != 2 || next.Slots["investment_amount"].Status != model.SlotStatusInvalid {
		t.Fatalf("retry state was not maintained: %#v", next)
	}
}

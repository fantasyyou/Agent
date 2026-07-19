package service

import (
	"agent-go/model"
	"time"
)

// DialogueStateService 由 Go 负责把 Python 的理解建议合并成可持久化任务状态。
type DialogueStateService struct{}

func NewDialogueStateService() *DialogueStateService { return &DialogueStateService{} }

// Apply 返回本轮合并后的状态；非 ask_user 动作代表当前任务结束，无需继续持久化。
func (s *DialogueStateService) Apply(current *model.DialogueState, decision model.DialogueDecision, now time.Time) (model.DialogueState, bool) {
	state := model.DialogueState{
		SchemaVersion: model.DialogueSchemaVersion,
		Workflow:      model.WorkflowState{Intent: decision.Intent, Version: model.WorkflowVersionV1, Status: model.WorkflowStatusCollecting},
		Slots:         map[string]model.SlotState{}, RetryCount: map[string]int{},
	}
	if current != nil {
		state = *current
		if state.Slots == nil {
			state.Slots = map[string]model.SlotState{}
		}
		if state.RetryCount == nil {
			state.RetryCount = map[string]int{}
		}
	}
	previousActiveSlot := state.ActiveSlot
	for name, update := range decision.SlotUpdates {
		update.Status = model.SlotStatusConfirmed
		if update.Source == "" {
			update.Source = model.SlotSourceUser
		}
		update.UpdatedAt = now.Format(time.RFC3339Nano)
		state.Slots[name] = update
		state.RetryCount[name] = 0
	}
	if previousActiveSlot != "" {
		if _, updated := decision.SlotUpdates[previousActiveSlot]; !updated && decision.NextAction == model.NextActionAskUser {
			state.RetryCount[previousActiveSlot]++
			slot := state.Slots[previousActiveSlot]
			slot.Status = model.SlotStatusInvalid
			slot.UpdatedAt = now.Format(time.RFC3339Nano)
			state.Slots[previousActiveSlot] = slot
		}
	}
	state.SchemaVersion = model.DialogueSchemaVersion
	state.Workflow = model.WorkflowState{Intent: decision.Intent, Version: model.WorkflowVersionV1, Status: model.WorkflowStatusCollecting}
	state.ActiveSlot = decision.ActiveSlot
	state.LastAction = decision.NextAction
	state.LastQuestion = decision.NextQuestion
	state.UpdatedAt = now.Format(time.RFC3339Nano)
	return state, decision.NextAction != model.NextActionAskUser
}

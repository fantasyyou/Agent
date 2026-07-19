package client

import (
	"agent-go/model"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestDialogueStateMapCanBeSerializedByProtobufStruct(t *testing.T) {
	state := model.DialogueState{
		SchemaVersion: model.DialogueSchemaVersion,
		Workflow:      model.WorkflowState{Intent: "product_recommendation", Version: "v1", Status: "collecting"},
		ActiveSlot:    "investment_amount",
		Slots:         map[string]model.SlotState{"risk_tolerance": {Value: "conservative", Status: model.SlotStatusConfirmed, Source: model.SlotSourceUser, Evidence: "偏保守", Confidence: 0.95}},
		RetryCount:    map[string]int{"investment_amount": 1},
		LastAction:    model.NextActionAskUser,
	}
	if _, err := structpb.NewStruct(dialogueStateMap(state)); err != nil {
		t.Fatalf("dialogue state must satisfy protobuf Struct serialization: %v", err)
	}
}

package redis

import (
	"agent-go/model"
	"testing"
)

func TestDecodeDialogueStateMigratesLegacySlots(t *testing.T) {
	state, err := decodeDialogueState([]byte(`{"intent":"product_recommendation","slots":{"risk_tolerance":"conservative"},"status":"need_clarification","last_question":"期限？"}`))
	if err != nil {
		t.Fatal(err)
	}
	if state.SchemaVersion != model.DialogueSchemaVersion || state.Workflow.Intent != "product_recommendation" {
		t.Fatalf("legacy workflow was not migrated: %#v", state)
	}
	if state.Slots["risk_tolerance"].Value != "conservative" || state.Slots["risk_tolerance"].Status != model.SlotStatusConfirmed {
		t.Fatalf("legacy slot was not migrated: %#v", state.Slots)
	}
}

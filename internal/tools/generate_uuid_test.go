package tools

import (
	"encoding/json"
	"regexp"
	"testing"
)

func TestGenerateUUIDCount(t *testing.T) {
	input := GenerateUUIDInput{Count: 3}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	var out GenerateUUIDOutput
	if err := json.Unmarshal([]byte(resp.Content[0].TextContent.Text), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(out.UUIDs) != 3 {
		t.Fatalf("expected 3 UUIDs, got %d", len(out.UUIDs))
	}
}

func TestGenerateUUIDUniqueness(t *testing.T) {
	input := GenerateUUIDInput{Count: 10}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	var out GenerateUUIDOutput
	if err := json.Unmarshal([]byte(resp.Content[0].TextContent.Text), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	seen := make(map[string]bool, len(out.UUIDs))
	for _, uid := range out.UUIDs {
		if seen[uid] {
			t.Fatalf("duplicate UUID: %s", uid)
		}
		seen[uid] = true
	}
}

func TestGenerateUUIDFormat(t *testing.T) {
	v4Pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	input := GenerateUUIDInput{Count: 20}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	var out GenerateUUIDOutput
	if err := json.Unmarshal([]byte(resp.Content[0].TextContent.Text), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	for _, uid := range out.UUIDs {
		if len(uid) != 36 {
			t.Fatalf("expected 36 chars, got %d for %s", len(uid), uid)
		}
		if !v4Pattern.MatchString(uid) {
			t.Fatalf("UUID %s does not match v4 format", uid)
		}
	}
}

func TestGenerateUUIDCountCap(t *testing.T) {
	input := GenerateUUIDInput{Count: 100}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	var out GenerateUUIDOutput
	if err := json.Unmarshal([]byte(resp.Content[0].TextContent.Text), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(out.UUIDs) != 50 {
		t.Fatalf("expected 50 UUIDs (capped), got %d", len(out.UUIDs))
	}
}

func TestGenerateUUIDPresetBPBasic(t *testing.T) {
	input := GenerateUUIDInput{Preset: "bp_basic"}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	var out GenerateUUIDOutput
	if err := json.Unmarshal([]byte(resp.Content[0].TextContent.Text), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out.PresetUsed != "bp_basic" {
		t.Fatalf("expected preset_used bp_basic, got %s", out.PresetUsed)
	}
	if len(out.Assignments) != 3 {
		t.Fatalf("expected 3 assignments, got %d", len(out.Assignments))
	}
	expectedKeys := []string{"header.uuid", "modules[0].uuid", "modules[1].uuid"}
	for _, k := range expectedKeys {
		if _, ok := out.Assignments[k]; !ok {
			t.Fatalf("missing assignment key: %s", k)
		}
	}
}

func TestGenerateUUIDPresetBPRPPair(t *testing.T) {
	input := GenerateUUIDInput{Preset: "bp_rp_pair"}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	var out GenerateUUIDOutput
	if err := json.Unmarshal([]byte(resp.Content[0].TextContent.Text), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out.PresetUsed != "bp_rp_pair" {
		t.Fatalf("expected preset_used bp_rp_pair, got %s", out.PresetUsed)
	}
	if len(out.Assignments) != 5 {
		t.Fatalf("expected 5 assignments, got %d", len(out.Assignments))
	}
}

func TestGenerateUUIDPresetScriptOnly(t *testing.T) {
	input := GenerateUUIDInput{Preset: "script_only"}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	var out GenerateUUIDOutput
	if err := json.Unmarshal([]byte(resp.Content[0].TextContent.Text), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out.PresetUsed != "script_only" {
		t.Fatalf("expected preset_used script_only, got %s", out.PresetUsed)
	}
	if len(out.Assignments) != 1 {
		t.Fatalf("expected 1 assignment, got %d", len(out.Assignments))
	}
	if _, ok := out.Assignments["script_module.uuid"]; !ok {
		t.Fatalf("missing assignment key: script_module.uuid")
	}
}

func TestGenerateUUIDInvalidPreset(t *testing.T) {
	input := GenerateUUIDInput{Preset: "nonexistent"}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	if len(resp.Content) == 0 {
		t.Fatal("expected error content, got empty response")
	}
	text := resp.Content[0].TextContent.Text
	if text != `Invalid preset "nonexistent". Valid presets: bp_basic, bp_rp_pair, script_only` {
		t.Fatalf("unexpected error message: %s", text)
	}
}

func TestGenerateUUIDExplicitSlots(t *testing.T) {
	slots := []string{"header.uuid", "modules[0].uuid"}
	input := GenerateUUIDInput{Slots: slots, Format: "assignments"}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	var out GenerateUUIDOutput
	if err := json.Unmarshal([]byte(resp.Content[0].TextContent.Text), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(out.Assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(out.Assignments))
	}
	for _, s := range slots {
		if _, ok := out.Assignments[s]; !ok {
			t.Fatalf("missing slot: %s", s)
		}
		if len(out.Assignments[s]) != 36 {
			t.Fatalf("expected 36-char UUID for slot %s, got %q", s, out.Assignments[s])
		}
	}
}

func TestGenerateUUIDFormatAssignments(t *testing.T) {
	input := GenerateUUIDInput{Preset: "bp_basic", Format: "assignments"}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	var out GenerateUUIDOutput
	if err := json.Unmarshal([]byte(resp.Content[0].TextContent.Text), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out.CopyPasteBlock == "" {
		t.Fatal("expected non-empty copy_paste_block")
	}
}

func TestGenerateUUIDFormatPlainNoSlots(t *testing.T) {
	input := GenerateUUIDInput{Count: 3, Format: "plain"}
	resp, err := handleGenerateUUID(input)
	if err != nil {
		t.Fatalf("handleGenerateUUID returned error: %v", err)
	}
	var out GenerateUUIDOutput
	if err := json.Unmarshal([]byte(resp.Content[0].TextContent.Text), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(out.Assignments) != 0 {
		t.Fatalf("expected empty assignments, got %d", len(out.Assignments))
	}
	if out.PresetUsed != "" {
		t.Fatalf("expected empty preset_used, got %s", out.PresetUsed)
	}
}

package tools

import (
	"encoding/json"
	"testing"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/apidiff"
)

func TestDiffInputStruct(t *testing.T) {
	input := DiffScriptAPIVersionsInput{
		Module:      "@minecraft/server",
		FromVersion: "2.7.0",
		ToVersion:   "2.8.0",
	}
	if input.Module != "@minecraft/server" {
		t.Errorf("unexpected Module: %s", input.Module)
	}
	if input.FromVersion != "2.7.0" {
		t.Errorf("unexpected FromVersion: %s", input.FromVersion)
	}
	if input.ToVersion != "2.8.0" {
		t.Errorf("unexpected ToVersion: %s", input.ToVersion)
	}
}

func TestDiffOutputFormat(t *testing.T) {
	result := &apidiff.DiffResult{
		Module:        "@minecraft/server",
		RequestedFrom: "2.7.0",
		RequestedTo:   "2.8.0",
		BreakingChanges: []apidiff.Change{
			{Kind: apidiff.Removed, Symbol: "someFunction", Details: "Symbol removed"},
		},
		NonBreaking: []apidiff.Change{
			{Kind: apidiff.Added, Symbol: "newFunction", Details: "Symbol added"},
		},
		Summary: "1 breaking, 1 non-breaking",
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal DiffResult: %v", err)
	}
	var decoded apidiff.DiffResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal DiffResult: %v", err)
	}
	if decoded.Module != "@minecraft/server" {
		t.Errorf("expected Module @minecraft/server, got %s", decoded.Module)
	}
	if len(decoded.BreakingChanges) != 1 {
		t.Errorf("expected 1 BreakingChange, got %d", len(decoded.BreakingChanges))
	}
	if decoded.BreakingChanges[0].Symbol != "someFunction" {
		t.Errorf("expected symbol 'someFunction', got %s", decoded.BreakingChanges[0].Symbol)
	}
}

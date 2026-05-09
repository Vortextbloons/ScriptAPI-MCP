package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/snippets"
)

func TestGenerateBedrockSnippetIntegrationJS(t *testing.T) {
	out, err := snippets.GenerateSnippet("beforeEvents.playerBreakBlock", "javascript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	output := string(b)

	if !strings.Contains(output, "import { world }") {
		t.Error("output missing world import")
	}
	if !strings.Contains(output, "playerBreakBlock") {
		t.Error("output missing playerBreakBlock")
	}
	if !strings.Contains(output, "required_modules") {
		t.Error("output missing required_modules")
	}
	if !strings.Contains(output, "files") {
		t.Error("output missing files")
	}
}

func TestGenerateBedrockSnippetIntegrationTS(t *testing.T) {
	out, err := snippets.GenerateSnippet("beforeEvents.playerBreakBlock", "typescript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	output := string(b)

	if !strings.Contains(output, "import type { BlockBreakAfterEvent }") {
		t.Error("TS output missing BlockBreakAfterEvent type import")
	}
	if !strings.Contains(output, ": void") {
		t.Error("TS output missing void return type")
	}
}

func TestGenerateBedrockSnippetModuleVersion(t *testing.T) {
	out, err := snippets.GenerateSnippet("afterEvents.playerSpawn", "javascript", "", "2.7.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := out.Files["src/main.js"]
	if !strings.Contains(content, "Target: @minecraft/server@2.7.0") {
		t.Error("output missing version comment")
	}
}

func TestGenerateBedrockSnippetAllTypes(t *testing.T) {
	types := []string{
		"beforeEvents.playerBreakBlock",
		"afterEvents.playerSpawn",
		"worldInitialize",
		"custom_item_template",
		"custom_block_template",
		"script_event_handler",
	}
	for _, st := range types {
		out, err := snippets.GenerateSnippet(st, "javascript", "", "", false)
		if err != nil {
			t.Fatalf("unexpected error for type %q: %v", st, err)
		}
		if len(out.Files) != 1 {
			t.Errorf("type %q: expected 1 file, got %d", st, len(out.Files))
		}
		if len(out.RequiredModules) != 1 || out.RequiredModules[0] != "@minecraft/server" {
			t.Errorf("type %q: expected required_modules [@minecraft/server]", st)
		}

		// Test TS variant too
		outTS, err := snippets.GenerateSnippet(st, "typescript", "", "", false)
		if err != nil {
			t.Fatalf("unexpected TS error for type %q: %v", st, err)
		}
		for _, content := range outTS.Files {
			if strings.Contains(content, "any") {
				t.Errorf("type %q TS should not contain 'any': %s", st, content)
			}
		}
	}
}

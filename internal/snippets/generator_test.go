package snippets

import (
	"strings"
	"testing"
)

func TestGenerateJSBeforeEvents(t *testing.T) {
	out, err := GenerateSnippet("beforeEvents.playerBreakBlock", "javascript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.js"]
	if !strings.Contains(src, `import { world }`) {
		t.Errorf("expected import { world }, got:\n%s", src)
	}
	if !strings.Contains(src, `.playerBreakBlock.subscribe`) {
		t.Errorf("expected .playerBreakBlock.subscribe, got:\n%s", src)
	}
	if !strings.Contains(src, `// your code here`) {
		t.Errorf("expected // your code here, got:\n%s", src)
	}
	if len(out.RequiredModules) != 1 || out.RequiredModules[0] != "@minecraft/server" {
		t.Errorf("expected RequiredModules [@minecraft/server], got %v", out.RequiredModules)
	}
}

func TestGenerateTSBeforeEvents(t *testing.T) {
	out, err := GenerateSnippet("beforeEvents.playerBreakBlock", "typescript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.ts"]
	if !strings.Contains(src, `import type { BlockBreakAfterEvent }`) {
		t.Errorf("expected import type { BlockBreakAfterEvent }, got:\n%s", src)
	}
	if !strings.Contains(src, `: BlockBreakAfterEvent`) {
		t.Errorf("expected : BlockBreakAfterEvent, got:\n%s", src)
	}
	if !strings.Contains(src, `: void =>`) {
		t.Errorf("expected : void =>, got:\n%s", src)
	}
}

func TestGenerateCustomItemJS(t *testing.T) {
	out, err := GenerateSnippet("custom_item_template", "javascript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.js"]
	if !strings.Contains(src, "your_namespace:your_component") {
		t.Errorf("expected default placeholder your_namespace:your_component, got:\n%s", src)
	}
}

func TestGenerateCustomItemTS(t *testing.T) {
	out, err := GenerateSnippet("custom_item_template", "typescript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.ts"]
	if !strings.Contains(src, `import type { WorldInitializeAfterEvent, ItemComponentUseEvent }`) {
		t.Errorf("expected type imports for WorldInitializeAfterEvent and ItemComponentUseEvent, got:\n%s", src)
	}
	if !strings.Contains(src, `: ItemComponentUseEvent`) {
		t.Errorf("expected : ItemComponentUseEvent, got:\n%s", src)
	}
	if !strings.Contains(src, `: void =>`) {
		t.Errorf("expected : void =>, got:\n%s", src)
	}
}

func TestGenerateWithName(t *testing.T) {
	out, err := GenerateSnippet("custom_item_template", "javascript", "my:custom_item", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.js"]
	if !strings.Contains(src, `"my:custom_item"`) {
		t.Errorf("expected name my:custom_item injected, got:\n%s", src)
	}
	if strings.Contains(src, "your_namespace:your_component") {
		t.Errorf("default placeholder should not appear when name is provided")
	}
}

func TestGenerateWithModuleVersion(t *testing.T) {
	out, err := GenerateSnippet("beforeEvents.playerBreakBlock", "javascript", "", "1.21.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.js"]
	if !strings.Contains(src, "// Target: @minecraft/server@1.21.0") {
		t.Errorf("expected version comment, got:\n%s", src)
	}
}

func TestGenerateScriptEventHandlerJS(t *testing.T) {
	out, err := GenerateSnippet("script_event_handler", "javascript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.js"]
	if !strings.Contains(src, `import { system }`) {
		t.Errorf("expected import { system }, got:\n%s", src)
	}
	if strings.Contains(src, `import { world }`) {
		t.Errorf("should NOT import world for script_event_handler, got:\n%s", src)
	}
	if !strings.Contains(src, `.scriptEventReceive.subscribe`) {
		t.Errorf("expected .scriptEventReceive.subscribe, got:\n%s", src)
	}
}

func TestGenerateScriptEventHandlerTS(t *testing.T) {
	out, err := GenerateSnippet("script_event_handler", "typescript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.ts"]
	if !strings.Contains(src, `import type { ScriptEventReceiveEvent }`) {
		t.Errorf("expected import type { ScriptEventReceiveEvent }, got:\n%s", src)
	}
	if !strings.Contains(src, `: ScriptEventReceiveEvent`) {
		t.Errorf("expected : ScriptEventReceiveEvent, got:\n%s", src)
	}
}

func TestGenerateUnknownType(t *testing.T) {
	_, err := GenerateSnippet("nonexistent_type", "javascript", "", "", false)
	if err == nil {
		t.Fatal("expected error for unknown type, got nil")
	}
}

func TestGenerateJavaScriptAllTypes(t *testing.T) {
	types := []string{
		"beforeEvents.playerBreakBlock",
		"afterEvents.playerSpawn",
		"worldInitialize",
		"custom_item_template",
		"custom_block_template",
		"script_event_handler",
	}
	if len(AllDefinitions) != 6 {
		t.Fatalf("expected 6 definitions, got %d", len(AllDefinitions))
	}
	for _, st := range types {
		_, err := GenerateSnippet(st, "javascript", "", "", false)
		if err != nil {
			t.Errorf("javascript variant for %s failed: %v", st, err)
		}
	}
}

func TestGenerateTypeScriptAllTypes(t *testing.T) {
	types := []string{
		"beforeEvents.playerBreakBlock",
		"afterEvents.playerSpawn",
		"worldInitialize",
		"custom_item_template",
		"custom_block_template",
		"script_event_handler",
	}
	for _, st := range types {
		out, err := GenerateSnippet(st, "typescript", "", "", false)
		if err != nil {
			t.Errorf("typescript variant for %s failed: %v", st, err)
			continue
		}
		src := out.Files["src/main.ts"]
		if !strings.Contains(src, `: void =>`) {
			t.Errorf("typescript variant for %s missing : void =>, got:\n%s", st, src)
		}
	}
}

func TestGenerateNoAnyInTS(t *testing.T) {
	types := []string{
		"beforeEvents.playerBreakBlock",
		"afterEvents.playerSpawn",
		"worldInitialize",
		"custom_item_template",
		"custom_block_template",
		"script_event_handler",
	}
	for _, st := range types {
		out, err := GenerateSnippet(st, "typescript", "", "", false)
		if err != nil {
			t.Errorf("typescript variant for %s failed: %v", st, err)
			continue
		}
		src := out.Files["src/main.ts"]
		if strings.Contains(src, " any") || strings.Contains(src, "any ") {
			t.Errorf("typescript variant for %s contains 'any' keyword:\n%s", st, src)
		}
	}
}

func TestGenerateSemicolonsInTS(t *testing.T) {
	types := []string{
		"beforeEvents.playerBreakBlock",
		"afterEvents.playerSpawn",
		"worldInitialize",
		"custom_item_template",
		"custom_block_template",
		"script_event_handler",
	}
	for _, st := range types {
		out, err := GenerateSnippet(st, "typescript", "", "", false)
		if err != nil {
			t.Errorf("typescript variant for %s failed: %v", st, err)
			continue
		}
		src := out.Files["src/main.ts"]
		if !strings.Contains(src, ";") {
			t.Errorf("typescript variant for %s missing semicolons:\n%s", st, src)
		}
	}
}

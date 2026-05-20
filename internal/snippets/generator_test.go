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

var originalEventTypes = []string{
	"beforeEvents.playerBreakBlock",
	"afterEvents.playerSpawn",
	"worldInitialize",
	"custom_item_template",
	"custom_block_template",
	"script_event_handler",
}

var advancedPatternTypes = []string{
	"runtime.plugin_registry",
	"runtime.background_scheduler",
	"runtime.profile_cache",
	"runtime.cooldown_manager",
	"ui.action_form_wizard",
	"interaction.item_interaction_handler",
	"storage.dynamic_property_store",
	"storage.world_config",
	"equipment.equipment_scanner",
	"item.lore_builder",
	"balance.scaled_value",
	"command.custom_slash_command",
}

var allTestTypes = append(originalEventTypes, advancedPatternTypes...)

func TestGenerateJavaScriptAllTypes(t *testing.T) {
	if len(AllDefinitions) != 18 {
		t.Fatalf("expected 18 definitions, got %d", len(AllDefinitions))
	}
	for _, st := range allTestTypes {
		_, err := GenerateSnippet(st, "javascript", "", "", false)
		if err != nil {
			t.Errorf("javascript variant for %s failed: %v", st, err)
		}
	}
}

func TestGenerateTypeScriptEventSnippets(t *testing.T) {
	for _, st := range originalEventTypes {
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

func TestGenerateTypeScriptAdvancedPatterns(t *testing.T) {
	for _, st := range advancedPatternTypes {
		out, err := GenerateSnippet(st, "typescript", "", "", false)
		if err != nil {
			t.Errorf("typescript variant for %s failed: %v", st, err)
			continue
		}
		src := out.Files["src/main.ts"]
		if !strings.Contains(src, ": void") && !strings.Contains(src, ">") {
			t.Errorf("typescript variant for %s missing type annotations or generics, got:\n%s", st, src)
		}
	}
}

// The advanced patterns use 'any' legitimately (Minecraft API getComponent() etc.)
// so only check the original event snippets for no-any requirement.
func TestGenerateNoAnyInOriginalTS(t *testing.T) {
	for _, st := range originalEventTypes {
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
	for _, st := range allTestTypes {
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

func TestGenerateMetadataOutput(t *testing.T) {
	out, err := GenerateSnippet("runtime.plugin_registry", "javascript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Type != "runtime.plugin_registry" {
		t.Errorf("expected Type runtime.plugin_registry, got %s", out.Type)
	}
	if out.Category != "runtime" {
		t.Errorf("expected Category runtime, got %s", out.Category)
	}
	if out.Complexity != "complex" {
		t.Errorf("expected Complexity complex, got %s", out.Complexity)
	}
	if len(out.Tags) == 0 {
		t.Errorf("expected non-empty Tags")
	}
	if len(out.RequiredModules) == 0 {
		t.Errorf("expected non-empty RequiredModules")
	}
}

func TestGenerateRTSnippetPath(t *testing.T) {
	out, err := GenerateSnippet("runtime.cooldown_manager", "javascript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.js"]
	if !strings.Contains(src, "system.currentTick") {
		t.Errorf("expected system.currentTick in output, got:\n%s", src)
	}
}

func TestGenerateUIFormSnippet(t *testing.T) {
	out, err := GenerateSnippet("ui.action_form_wizard", "typescript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.ts"]
	if !strings.Contains(src, "ActionFormData") {
		t.Errorf("expected ActionFormData in output, got:\n%s", src)
	}
	if !strings.Contains(src, "@minecraft/server-ui") {
		t.Errorf("expected @minecraft/server-ui import, got:\n%s", src)
	}
}

func TestGenerateEquipmentSnippet(t *testing.T) {
	out, err := GenerateSnippet("equipment.equipment_scanner", "javascript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.js"]
	if !strings.Contains(src, "scanEquipment") {
		t.Errorf("expected scanEquipment in output, got:\n%s", src)
	}
}

func TestGenerateBalanceSnippet(t *testing.T) {
	out, err := GenerateSnippet("balance.scaled_value", "javascript", "", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	src := out.Files["src/main.js"]
	if !strings.Contains(src, "rollWeightedLevel") {
		t.Errorf("expected rollWeightedLevel in output, got:\n%s", src)
	}
}

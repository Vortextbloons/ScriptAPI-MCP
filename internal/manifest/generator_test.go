package manifest

import (
	"reflect"
	"testing"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/models"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

// TestGenerateUUID tests UUID generation
func TestGenerateUUID(t *testing.T) {
	uuid1 := GenerateUUID()
	uuid2 := GenerateUUID()

	if uuid1 == "" {
		t.Error("GenerateUUID returned empty string")
	}
	if uuid1 == uuid2 {
		t.Error("GenerateUUID returned duplicate UUIDs")
	}
	if len(uuid1) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid1))
	}
}

// TestGenerateBP tests behavior pack manifest generation
func TestGenerateBP(t *testing.T) {
	deps := []models.Dependency{
		{ModuleName: "@minecraft/server-ui", Version: "2.1.0-beta"},
		{ModuleName: "@minecraft/server", Version: "2.7.0-beta"},
	}
	bp := GenerateBP("Test Addon", "A test addon", deps, "test-uuid")

	if bp.Header.Name != "Test Addon" {
		t.Errorf("name = %q, want Test Addon", bp.Header.Name)
	}
	if bp.Header.Description != "A test addon" {
		t.Errorf("description = %q, want A test addon", bp.Header.Description)
	}
	if bp.Header.UUID != "test-uuid" {
		t.Errorf("uuid = %q, want test-uuid", bp.Header.UUID)
	}
	if len(bp.Dependencies) != 2 {
		t.Errorf("dependencies count = %d, want 2", len(bp.Dependencies))
	}
	if bp.Dependencies[0].ModuleName != "@minecraft/server" {
		t.Errorf("dependency module = %q, want @minecraft/server", bp.Dependencies[0].ModuleName)
	}
	if !reflect.DeepEqual([]string{bp.Dependencies[0].ModuleName, bp.Dependencies[1].ModuleName}, []string{"@minecraft/server", "@minecraft/server-ui"}) {
		t.Errorf("dependencies not sorted: %+v", bp.Dependencies)
	}
	if len(bp.Modules) != 2 {
		t.Errorf("modules count = %d, want 2", len(bp.Modules))
	}
}

// TestGenerateRP tests resource pack manifest generation
func TestGenerateRP(t *testing.T) {
	rp := GenerateRP("Test Addon", "A test addon", "rp-uuid", "bp-uuid")

	if rp.Header.Name != "Test Addon RP" {
		t.Errorf("name = %q, want Test Addon RP", rp.Header.Name)
	}
	if rp.Header.UUID != "rp-uuid" {
		t.Errorf("uuid = %q, want rp-uuid", rp.Header.UUID)
	}
	if len(rp.Dependencies) != 1 {
		t.Errorf("dependencies count = %d, want 1", len(rp.Dependencies))
	}
	if rp.Dependencies[0].UUID != "bp-uuid" {
		t.Errorf("dependency uuid = %q, want bp-uuid", rp.Dependencies[0].UUID)
	}
	if len(rp.Modules) != 1 {
		t.Errorf("modules count = %d, want 1", len(rp.Modules))
	}
}

// TestBuildDependencies tests basic dependency building
func TestBuildDependencies(t *testing.T) {
	deps := BuildDependencies("2.7.0-beta", true)

	if len(deps) != 2 {
		t.Errorf("dependencies count = %d, want 2", len(deps))
	}
	if deps[0].ModuleName != "@minecraft/server" {
		t.Errorf("first dep = %q, want @minecraft/server", deps[0].ModuleName)
	}
	if deps[1].ModuleName != "@minecraft/server-ui" {
		t.Errorf("second dep = %q, want @minecraft/server-ui", deps[1].ModuleName)
	}
}

// TestBuildDependenciesNoUI tests dependency building without UI
func TestBuildDependenciesNoUI(t *testing.T) {
	deps := BuildDependencies("1.0.0", false)

	if len(deps) != 1 {
		t.Errorf("dependencies count = %d, want 1", len(deps))
	}
	if deps[0].ModuleName != "@minecraft/server" {
		t.Errorf("first dep = %q, want @minecraft/server", deps[0].ModuleName)
	}
}

// TestUpdateDependencies tests adding and removing dependencies
func TestUpdateDependencies(t *testing.T) {
	manifest := &models.Manifest{
		Dependencies: []models.Dependency{
			{ModuleName: "@minecraft/server", Version: "1.0.0"},
		},
	}

	// Add a module
	err := UpdateDependencies(manifest, []string{"@minecraft/server-ui"}, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Dependencies) != 2 {
		t.Errorf("dependencies count = %d, want 2", len(manifest.Dependencies))
	}

	// Remove a module
	err = UpdateDependencies(manifest, []string{}, []string{"@minecraft/server"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Dependencies) != 1 {
		t.Errorf("dependencies count = %d, want 1", len(manifest.Dependencies))
	}
}

// TestUpdateDependenciesInvalidModule tests validation
func TestUpdateDependenciesInvalidModule(t *testing.T) {
	manifest := &models.Manifest{
		Dependencies: []models.Dependency{},
	}

	err := UpdateDependencies(manifest, []string{"invalid-module"}, []string{})
	if err == nil {
		t.Error("expected error for invalid module")
	}
}

// TestFormatManifest tests manifest formatting
func TestFormatManifest(t *testing.T) {
	manifest := models.Manifest{
		FormatVersion: 2,
		Header: models.ManifestHeader{
			Name:    "Test",
			Version: []int{1, 0, 0},
		},
	}

	json, err := FormatManifest(manifest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if json == "" {
		t.Error("FormatManifest returned empty string")
	}
	if !containsString(json, "Test") {
		t.Error("formatted JSON should contain 'Test'")
	}
}

// TestParseManifest tests manifest parsing
func TestParseManifest(t *testing.T) {
	json := `{
		"format_version": 2,
		"header": {
			"name": "Test",
			"version": [1, 0, 0]
		}
	}`

	manifest, err := ParseManifest(json)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manifest.FormatVersion != 2 {
		t.Errorf("format_version = %d, want 2", manifest.FormatVersion)
	}
	if manifest.Header.Name != "Test" {
		t.Errorf("name = %q, want Test", manifest.Header.Name)
	}
}

// TestParseManifestInvalidJSON tests error handling
func TestParseManifestInvalidJSON(t *testing.T) {
	_, err := ParseManifest("invalid json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestFileStructure tests file structure generation
func TestFileStructure(t *testing.T) {
	files := FileStructure("MyAddon", false, "javascript", false)

	if len(files) != 6 {
		t.Errorf("file count = %d, want 6", len(files))
	}
	if !contains(files, "behavior_pack/manifest.json") {
		t.Error("should contain behavior_pack/manifest.json")
	}
	if !contains(files, "behavior_pack/pack_icon.png") {
		t.Error("should contain pack_icon.png")
	}
	if !contains(files, "src/main.js") {
		t.Error("should contain src/main.js")
	}
}

// TestFileStructureWithRP tests file structure with resource pack
func TestFileStructureWithRP(t *testing.T) {
	files := FileStructure("MyAddon", true, "javascript", false)

	if len(files) != 9 {
		t.Errorf("file count = %d, want 9", len(files))
	}
	if !contains(files, "resource_pack/manifest.json") {
		t.Error("should contain resource_pack/manifest.json")
	}
	if !contains(files, "resource_pack/pack_icon.png") {
		t.Error("should contain resource_pack/pack_icon.png")
	}
}

// TestFileStructureTypeScript tests file structure with TypeScript
func TestFileStructureTypeScript(t *testing.T) {
	files := FileStructure("MyAddon", false, "typescript", false)

	if len(files) != 7 {
		t.Errorf("file count = %d, want 7", len(files))
	}
	if !contains(files, "tsconfig.json") {
		t.Error("should contain tsconfig.json")
	}
	if !contains(files, "src/main.ts") {
		t.Error("should contain src/main.ts")
	}
}

// TestFileStructureWithDeploy tests file structure with deploy script enabled
func TestFileStructureWithDeploy(t *testing.T) {
	files := FileStructure("MyAddon", false, "javascript", true)

	if len(files) != 9 {
		t.Errorf("file count = %d, want 9", len(files))
	}
	if !contains(files, "package.json") {
		t.Error("should contain package.json")
	}
	if !contains(files, "scripts/deploy.js") {
		t.Error("should contain scripts/deploy.js")
	}
}

// TestGenerateStarterCode tests starter code generation
func TestGenerateStarterCode(t *testing.T) {
	code := GenerateStarterCode("javascript", "1.21.60")

	if len(code) != 1 {
		t.Errorf("file count = %d, want 1", len(code))
	}
	if _, ok := code["src/main.js"]; !ok {
		t.Error("should contain src/main.js")
	}
}

// TestGenerateStarterCodeTypeScript tests TypeScript starter code
func TestGenerateStarterCodeTypeScript(t *testing.T) {
	code := GenerateStarterCode("typescript", "latest")

	if len(code) != 2 {
		t.Errorf("file count = %d, want 2", len(code))
	}
	if _, ok := code["src/main.ts"]; !ok {
		t.Error("should contain src/main.ts")
	}
	if _, ok := code["tsconfig.json"]; !ok {
		t.Error("should contain tsconfig.json")
	}
}

// TestBuildDependenciesWithChannel tests version resolution with channel
func TestBuildDependenciesWithChannel(t *testing.T) {
	// Create a mock client
	client := npm.NewClient()

	// Use a specific version that should exist
	deps, err := BuildDependenciesWithChannel(client, []string{"@minecraft/server"}, "latest", "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("dependencies count = %d, want 1", len(deps))
	}
	if deps[0].ModuleName != "@minecraft/server" {
		t.Errorf("module = %q, want @minecraft/server", deps[0].ModuleName)
	}
	if deps[0].Version == "" {
		t.Error("version should not be empty")
	}
}

// TestBuildDependenciesWithChannelDefaultModules tests default module handling
func TestBuildDependenciesWithChannelDefaultModules(t *testing.T) {
	client := npm.NewClient()

	deps, err := BuildDependenciesWithChannel(client, []string{}, "latest", "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("dependencies count = %d, want 1", len(deps))
	}
	if deps[0].ModuleName != "@minecraft/server" {
		t.Errorf("module = %q, want @minecraft/server", deps[0].ModuleName)
	}
}

// TestBuildDependenciesWithChannelStable tests stable channel
func TestBuildDependenciesWithChannelStable(t *testing.T) {
	client := npm.NewClient()

	deps, err := BuildDependenciesWithChannel(client, []string{"@minecraft/server"}, "latest", "stable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("dependencies count = %d, want 1", len(deps))
	}
	// Stable versions should not contain "beta"
	if containsString(deps[0].Version, "beta") {
		t.Errorf("stable version %q should not contain 'beta'", deps[0].Version)
	}
}

// TestBuildDependenciesWithChannelMultipleModules tests multiple module resolution
func TestBuildDependenciesWithChannelMultipleModules(t *testing.T) {
	client := npm.NewClient()

	deps, err := BuildDependenciesWithChannel(client, []string{
		"@minecraft/server",
		"@minecraft/server-ui",
	}, "latest", "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("dependencies count = %d, want 2", len(deps))
	}
}

// Helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsInternal(s, substr))
}

func containsInternal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestBuildDependenciesWithValidation tests validation against node_modules
func TestBuildDependenciesWithValidation(t *testing.T) {
	client := npm.NewClient()

	// This will likely return warnings since node_modules probably doesn't exist in test env
	deps, warnings, err := BuildDependenciesWithValidation(".", client, []string{"@minecraft/server"}, "latest", "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(deps) != 1 {
		t.Errorf("dependencies count = %d, want 1", len(deps))
	}
	// Should have warnings since node_modules/@minecraft/server likely doesn't exist
	if len(warnings) == 0 {
		t.Log("No warnings - node_modules might exist or validation skipped")
	}
}

// TestUpdateDependenciesDeprecatedModule tests deprecated module rejection
func TestUpdateDependenciesDeprecatedModule(t *testing.T) {
	manifest := &models.Manifest{
		Dependencies: []models.Dependency{},
	}

	err := UpdateDependencies(manifest, []string{"mojang-minecraft"}, []string{})
	if err == nil {
		t.Error("expected error for deprecated module")
	}
}

// TestManifestHeaderMinEngineVersion tests min engine version
func TestManifestHeaderMinEngineVersion(t *testing.T) {
	deps := []models.Dependency{
		{ModuleName: "@minecraft/server", Version: "1.0.0"},
	}
	bp := GenerateBP("Test", "Test desc", deps, "uuid")

	expected := []int{1, 21, 60}
	if len(bp.Header.MinEngineVersion) != len(expected) {
		t.Errorf("min_engine_version length = %d, want %d", len(bp.Header.MinEngineVersion), len(expected))
	}
	for i, v := range expected {
		if bp.Header.MinEngineVersion[i] != v {
			t.Errorf("min_engine_version[%d] = %d, want %d", i, bp.Header.MinEngineVersion[i], v)
		}
	}
}

// TestGenerateUUIDFormat tests UUID format
func TestGenerateUUIDFormat(t *testing.T) {
	uuid := GenerateUUID()

	// Check format: 8-4-4-4-12 hex characters
	if len(uuid) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid))
	}

	// Check dashes at positions 8, 13, 18, 23
	for _, pos := range []int{8, 13, 18, 23} {
		if uuid[pos] != '-' {
			t.Errorf("UUID missing dash at position %d", pos)
		}
	}
}

// TestGenerateStarterCodeVersionInjection tests version in starter code
func TestGenerateStarterCodeVersionInjection(t *testing.T) {
	code := GenerateStarterCode("javascript", "1.21.60")
	jsCode := code["src/main.js"]

	if !containsString(jsCode, "1.21.60") {
		t.Error("starter code should contain the version string")
	}
}

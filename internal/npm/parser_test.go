package npm

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCompareSemver tests the semver comparison function
func TestCompareSemver(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"equal", "1.0.0", "1.0.0", 0},
		{"a > b", "2.6.0", "2.5.0", 1},
		{"a < b", "2.5.0", "2.6.0", -1},
		{"major diff", "2.0.0", "1.99.99", 1},
		{"stable over beta same base", "2.6.0", "2.6.0-beta.1.26.30-preview.21", 1},
		{"beta under stable same base", "2.6.0-beta.1.26.30-preview.21", "2.6.0", -1},
		{"longer base wins", "2.10.0", "2.9.0", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareSemver(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("compareSemver(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestNormalizeVersion tests version normalization with REAL npm formats
func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2.9.0-beta.1.26.30-preview.21", "2.9.0-beta"},
		{"2.8.0-beta.1.26.20-preview.28", "2.8.0-beta"},
		{"2.6.0", "2.6.0"},
		{"2.5.0", "2.5.0"},
		{"2.8.0-rc.1.26.30-preview.21", "2.8.0"},
		{"1.0.0-preview.1.19.60.22", "1.0.0"},
		{"2.9.0-beta", "2.9.0-beta"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestResolveVersionForChannelLatestStable tests stable resolution for "latest"
func TestResolveVersionForChannelLatestStable(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.5.0",
			"2.9.0-beta.1.26.30-preview.21",
			"2.8.0-beta.1.26.20-preview.28",
			"2.8.0-rc.1.26.30-preview.21",
			"2.4.0",
		},
		Tags: map[string]string{
			"latest": "2.6.0",
			"beta":   "2.9.0-beta.1.26.30-preview.21",
		},
	}

	result, err := ResolveVersionForChannel(vm, "latest", "stable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "2.6.0"
	if result != expected {
		t.Errorf("ResolveVersionForChannel(latest, stable) = %q, want %q", result, expected)
	}
}

// TestResolveVersionForChannelLatestBeta tests beta resolution for "latest"
// Beta channel should prefer versions with -beta but NOT -preview
func TestResolveVersionForChannelLatestBeta(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.9.0-beta.1.26.30",
			"2.8.0-beta.1.26.20",
			"2.9.0-beta.1.26.30-preview.21",
			"2.8.0-beta.1.26.20-preview.28",
		},
		Tags: map[string]string{
			"latest": "2.6.0",
			"beta":   "2.9.0-beta.1.26.30-preview.21",
		},
	}

	result, err := ResolveVersionForChannel(vm, "latest", "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should pick highest beta WITHOUT preview: 2.9.0-beta.1.26.30
	expected := "2.9.0-beta"
	if result != expected {
		t.Errorf("ResolveVersionForChannel(latest, beta) = %q, want %q", result, expected)
	}
}

// TestResolveVersionForChannelLatestPreview tests preview resolution for "latest"
func TestResolveVersionForChannelLatestPreview(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.9.0-beta.1.26.30-preview.21",
			"2.8.0-beta.1.26.20-preview.28",
			"2.8.0-rc.1.26.30-preview.21",
			"2.9.0-beta.1.26.30",
		},
		Tags: map[string]string{
			"latest": "2.6.0",
			"beta":   "2.9.0-beta.1.26.30-preview.21",
		},
	}

	result, err := ResolveVersionForChannel(vm, "latest", "preview")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should pick highest preview: 2.9.0-beta.1.26.30-preview.21
	expected := "2.9.0-beta"
	if result != expected {
		t.Errorf("ResolveVersionForChannel(latest, preview) = %q, want %q", result, expected)
	}
}

// TestResolveVersionForChannelSpecificStable tests specific version stable resolution
func TestResolveVersionForChannelSpecificStable(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.9.0-beta.1.26.30-preview.21",
			"2.5.0",
			"2.8.0-beta.1.26.20-preview.28",
		},
	}

	// When searching for stable with no matching stable versions, falls back to all
	// Here 1.26.30 is not in any stable version, so it falls back
	result, err := ResolveVersionForChannel(vm, "2.6", "stable")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2.6.0 contains "2.6" and is stable
	expected := "2.6.0"
	if result != expected {
		t.Errorf("ResolveVersionForChannel(2.6, stable) = %q, want %q", result, expected)
	}
}

// TestResolveVersionForChannelSpecificBeta tests specific version beta resolution
func TestResolveVersionForChannelSpecificBeta(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.9.0-beta.1.26.30",
			"2.8.0-beta.1.26.20",
			"2.9.0-beta.1.26.30-preview.21",
			"2.5.0",
		},
	}

	result, err := ResolveVersionForChannel(vm, "1.26.30", "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return beta version containing 1.26.30 (but not preview)
	expected := "2.9.0-beta"
	if result != expected {
		t.Errorf("ResolveVersionForChannel(1.26.30, beta) = %q, want %q", result, expected)
	}
}

// TestResolveVersionForChannelSpecificPreview tests specific version preview resolution
func TestResolveVersionForChannelSpecificPreview(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.9.0-beta.1.26.30-preview.21",
			"2.8.0-beta.1.26.20-preview.28",
			"2.8.0-rc.1.26.30-preview.21",
		},
	}

	result, err := ResolveVersionForChannel(vm, "1.26.30", "preview")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return preview version containing 1.26.30
	expected := "2.9.0-beta"
	if result != expected {
		t.Errorf("ResolveVersionForChannel(1.26.30, preview) = %q, want %q", result, expected)
	}
}

// TestResolveVersionForChannelDeterministic tests that same input always returns same output
func TestResolveVersionForChannelDeterministic(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.5.0",
			"2.9.0-beta.1.26.30-preview.21",
			"2.8.0-beta.1.26.20-preview.28",
			"2.9.0-beta.1.26.30",
			"2.4.0",
		},
		Tags: map[string]string{
			"latest": "2.6.0",
			"beta":   "2.9.0-beta.1.26.30-preview.21",
		},
	}

	// Run multiple times to check determinism
	for i := 0; i < 10; i++ {
		result1, err := ResolveVersionForChannel(vm, "latest", "stable")
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}

		result2, err := ResolveVersionForChannel(vm, "latest", "stable")
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}

		if result1 != result2 {
			t.Errorf("non-deterministic result: got %q and %q", result1, result2)
		}
	}
}

// TestResolveModuleVersions tests multi-module resolution
func TestResolveModuleVersions(t *testing.T) {
	matrices := map[string]*VersionMatrix{
		"@minecraft/server": {
			Module: "@minecraft/server",
			Versions: []string{
				"2.6.0",
				"2.9.0-beta.1.26.30-preview.21",
				"2.9.0-beta.1.26.30",
			},
			Tags: map[string]string{
				"latest": "2.6.0",
				"beta":   "2.9.0-beta.1.26.30-preview.21",
			},
		},
		"@minecraft/server-ui": {
			Module: "@minecraft/server-ui",
			Versions: []string{
				"2.1.0",
				"2.2.0-beta.1.26.30-preview.21",
				"2.2.0-beta.1.26.30",
			},
			Tags: map[string]string{
				"latest": "2.1.0",
				"beta":   "2.2.0-beta.1.26.30-preview.21",
			},
		},
	}

	results, err := ResolveModuleVersions(matrices, "latest", "beta")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results["@minecraft/server"] != "2.9.0-beta" {
		t.Errorf("server version = %q, want 2.9.0-beta", results["@minecraft/server"])
	}
	if results["@minecraft/server-ui"] != "2.2.0-beta" {
		t.Errorf("server-ui version = %q, want 2.2.0-beta", results["@minecraft/server-ui"])
	}
}

// TestResolveVersionOld tests the older ResolveVersion function
func TestResolveVersionOld(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.9.0-beta.1.26.30-preview.21",
			"2.5.0",
			"2.8.0-beta.1.26.20-preview.28",
		},
		Tags: map[string]string{
			"latest": "2.6.0",
			"beta":   "2.9.0-beta.1.26.30-preview.21",
		},
	}

	result, err := ResolveVersion(vm, "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "2.6.0" {
		t.Errorf("ResolveVersion(latest) = %q, want 2.6.0", result)
	}
}

// TestExtractTypes tests type extraction
func TestExtractTypes(t *testing.T) {
	dts := []byte(`
export declare class World {
    getDimension(id: string): Dimension;
}

export declare class Dimension {
    getEntities(): Entity[];
}

export declare interface Entity {
    id: string;
}
`)

	result, err := ExtractTypes(dts, "World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !containsSubstring(result, "class World") {
		t.Error("result should contain class World")
	}
	if !containsSubstring(result, "class Dimension") {
		t.Error("result should contain class Dimension (referenced type)")
	}
}

// TestNormalizeVersionRC tests RC versions are NOT normalized to beta
func TestNormalizeVersionRC(t *testing.T) {
	result := NormalizeVersion("2.8.0-rc.1.26.30-preview.21")
	if result != "2.8.0" {
		t.Errorf("NormalizeVersion(rc) = %q, want 2.8.0", result)
	}
}

// TestNormalizeVersionPreview tests preview versions are NOT normalized to beta
func TestNormalizeVersionPreview(t *testing.T) {
	result := NormalizeVersion("1.0.0-preview.1.19.60.22")
	if result != "1.0.0" {
		t.Errorf("NormalizeVersion(preview) = %q, want 1.0.0", result)
	}
}

// TestResolveVersionForChannelNoStableAvailable tests error handling
func TestResolveVersionForChannelNoStableAvailable(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.9.0-beta.1.26.30-preview.21",
			"2.8.0-beta.1.26.20-preview.28",
		},
	}

	_, err := ResolveVersionForChannel(vm, "latest", "stable")
	if err == nil {
		t.Error("expected error when no stable version exists, got nil")
	}
}

// TestCompareSemverEdgeCases tests edge cases for semver comparison
func TestCompareSemverEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{"major diff", "2.0.0", "1.99.99", 1},
		{"minor diff", "2.10.0", "2.9.0", 1},
		{"patch diff", "2.6.1", "2.6.0", 1},
		{"same with prerelease", "2.9.0-beta.1", "2.9.0-beta.1", 0},
		{"stable vs beta same base", "2.6.0", "2.6.0-beta.1.26.30", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareSemver(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("compareSemver(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestResolveVersionDeterministicOld tests older resolver determinism
func TestResolveVersionDeterministicOld(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.9.0-beta.1.26.30-preview.21",
			"2.5.0",
			"2.8.0-beta.1.26.20-preview.28",
			"2.4.0",
		},
		Tags: map[string]string{
			"latest": "2.6.0",
			"beta":   "2.9.0-beta.1.26.30-preview.21",
		},
	}

	for i := 0; i < 10; i++ {
		result1, err := ResolveVersion(vm, "latest")
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		result2, err := ResolveVersion(vm, "latest")
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		if result1 != result2 {
			t.Errorf("non-deterministic: %q vs %q", result1, result2)
		}
	}
}

// TestGetInstalledModule tests reading from node_modules
func TestGetInstalledModule(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "node_modules", "@minecraft", "server")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	packageJSON := `{"name":"@minecraft/server","version":"2.7.0-beta.1.26.14-stable"}`
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	installed, err := GetInstalledModule(tmpDir, "@minecraft/server")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if installed.Version != "2.7.0-beta.1.26.14-stable" {
		t.Errorf("version = %q, want 2.7.0-beta.1.26.14-stable", installed.Version)
	}
	if !installed.IsBeta {
		t.Error("expected IsBeta to be true")
	}
}

// TestValidateInstalledModules tests version validation
func TestValidateInstalledModules(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "node_modules", "@minecraft", "server")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	packageJSON := `{"name":"@minecraft/server","version":"2.7.0-beta.1.26.14-stable"}`
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	manifestVersions := map[string]string{
		"@minecraft/server": "2.7.0-beta",
	}

	warnings, err := ValidateInstalledModules(tmpDir, manifestVersions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got: %v", warnings)
	}

	manifestVersions = map[string]string{
		"@minecraft/server": "2.6.0",
	}

	warnings, err = ValidateInstalledModules(tmpDir, manifestVersions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) == 0 {
		t.Error("expected warning for version mismatch, got none")
	}
}

// TestResolveVersionsFromInstalled tests resolving from installed modules
func TestResolveVersionsFromInstalled(t *testing.T) {
	tmpDir := t.TempDir()

	serverDir := filepath.Join(tmpDir, "node_modules", "@minecraft", "server")
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "package.json"), []byte(`{"name":"@minecraft/server","version":"2.7.0-beta.1.26.14-stable"}`), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	uiDir := filepath.Join(tmpDir, "node_modules", "@minecraft", "server-ui")
	if err := os.MkdirAll(uiDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "package.json"), []byte(`{"name":"@minecraft/server-ui","version":"2.1.0-beta.1.26.14-stable"}`), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	resolved, err := ResolveVersionsFromInstalled(tmpDir, []string{"@minecraft/server", "@minecraft/server-ui"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved["@minecraft/server"] != "2.7.0-beta" {
		t.Errorf("server version = %q, want 2.7.0-beta", resolved["@minecraft/server"])
	}
	if resolved["@minecraft/server-ui"] != "2.1.0-beta" {
		t.Errorf("server-ui version = %q, want 2.1.0-beta", resolved["@minecraft/server-ui"])
	}
}

// TestGetInstalledMinecraftModules tests scanning for all installed modules
func TestGetInstalledMinecraftModules(t *testing.T) {
	tmpDir := t.TempDir()

	modules := []string{"server", "server-ui", "server-net"}
	for _, mod := range modules {
		dir := filepath.Join(tmpDir, "node_modules", "@minecraft", mod)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dirs: %v", err)
		}
		pkg := `{"name":"@minecraft/` + mod + `","version":"1.0.0"}`
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
			t.Fatalf("failed to write package.json: %v", err)
		}
	}

	installed, err := GetInstalledMinecraftModules(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(installed) != 3 {
		t.Errorf("expected 3 modules, got %d", len(installed))
	}
}

// TestReadProjectDependencies tests reading package.json
func TestReadProjectDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	packageJSON := `{
		"name": "test-project",
		"version": "1.0.0",
		"dependencies": {
			"@minecraft/server": "^2.7.0-beta"
		},
		"devDependencies": {
			"typescript": "^5.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	pkg, err := ReadProjectDependencies(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pkg.Name != "test-project" {
		t.Errorf("name = %q, want test-project", pkg.Name)
	}
	if pkg.Dependencies["@minecraft/server"] != "^2.7.0-beta" {
		t.Errorf("server dep = %q, want ^2.7.0-beta", pkg.Dependencies["@minecraft/server"])
	}
}

// TestResolveVersionForChannelEmptyVersionMatrix tests error handling
func TestResolveVersionForChannelEmptyVersionMatrix(t *testing.T) {
	vm := &VersionMatrix{
		Module:   "@minecraft/server",
		Versions: []string{},
		Tags:     map[string]string{},
	}

	_, err := ResolveVersionForChannel(vm, "latest", "beta")
	if err == nil {
		t.Error("expected error for empty version matrix")
	}
}

// TestResolveVersionForChannelInvalidVersion tests error handling
func TestResolveVersionForChannelInvalidVersion(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
		},
	}

	_, err := ResolveVersionForChannel(vm, "9.99.99", "stable")
	if err == nil {
		t.Error("expected error for non-existent version")
	}
}

// TestNormalizeVersionEmpty tests empty string
func TestNormalizeVersionEmpty(t *testing.T) {
	result := NormalizeVersion("")
	if result != "" {
		t.Errorf("NormalizeVersion(\"\") = %q, want empty string", result)
	}
}

// TestCompareSemverEmptyStrings tests empty string comparison
func TestCompareSemverEmptyStrings(t *testing.T) {
	result := compareSemver("", "")
	if result != 0 {
		t.Errorf("compareSemver(\"\", \"\") = %d, want 0", result)
	}
}

// TestResolveVersionForChannelNoBetaAvailable tests fallback when no beta exists
func TestResolveVersionForChannelNoBetaAvailable(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.5.0",
		},
	}

	_, err := ResolveVersionForChannel(vm, "latest", "beta")
	if err == nil {
		t.Error("expected error when no beta version exists")
	}
}

// TestResolveVersionForChannelNoPreviewAvailable tests fallback when no preview exists
func TestResolveVersionForChannelNoPreviewAvailable(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.9.0-beta.1.26.30",
		},
	}

	_, err := ResolveVersionForChannel(vm, "latest", "preview")
	if err == nil {
		t.Error("expected error when no preview version exists")
	}
}

// TestMatchesChannel tests channel matching logic
func TestMatchesChannel(t *testing.T) {
	tests := []struct {
		version  string
		channel  string
		expected bool
	}{
		{"2.6.0", "stable", true},
		{"2.9.0-beta.1.26.30", "stable", false},
		{"2.9.0-beta.1.26.30", "beta", true},
		{"2.9.0-beta.1.26.30-preview.21", "beta", false},
		{"2.9.0-beta.1.26.30-preview.21", "preview", true},
		{"2.8.0-rc.1.26.30-preview.21", "preview", true},
		{"2.8.0-rc.1.26.30-preview.21", "stable", false},
		{"2.8.0-rc.1.26.30-preview.21", "beta", false},
	}

	for _, tt := range tests {
		t.Run(tt.version+"_"+tt.channel, func(t *testing.T) {
			result := matchesChannel(tt.version, tt.channel)
			if result != tt.expected {
				t.Errorf("matchesChannel(%q, %q) = %v, want %v", tt.version, tt.channel, result, tt.expected)
			}
		})
	}
}

// TestResolveVersionForChannelSpecificMCVersion tests matching specific MC version
func TestResolveVersionForChannelSpecificMCVersion(t *testing.T) {
	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.9.0-beta.1.26.30-preview.21",
			"2.8.0-beta.1.26.20-preview.28",
			"2.8.0-beta.1.26.20-preview.27",
			"2.6.0",
		},
	}

	result, err := ResolveVersionForChannel(vm, "1.26.30", "preview")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "2.9.0-beta"
	if result != expected {
		t.Errorf("ResolveVersionForChannel(1.26.30, preview) = %q, want %q", result, expected)
	}
}

// TestCacheClear tests cache clearing
func TestCacheClear(t *testing.T) {
	cache := NewCache()
	cache.Set("test", []byte("data"), 1000000)

	_, ok := cache.Get("test")
	if !ok {
		t.Fatal("expected cache hit")
	}

	cache.Clear()

	_, ok = cache.Get("test")
	if ok {
		t.Error("expected cache miss after clear")
	}
}

// TestCacheDelete tests cache deletion
func TestCacheDelete(t *testing.T) {
	cache := NewCache()
	cache.Set("test1", []byte("data1"), 1000000)
	cache.Set("test2", []byte("data2"), 1000000)

	cache.Delete("test1")

	_, ok := cache.Get("test1")
	if ok {
		t.Error("expected test1 to be deleted")
	}

	_, ok = cache.Get("test2")
	if !ok {
		t.Error("expected test2 to still exist")
	}
}

// Helper functions
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

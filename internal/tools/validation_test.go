package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateInstalledDependencies(t *testing.T) {
	tmpDir := t.TempDir()
	moduleDir := filepath.Join(tmpDir, "node_modules", "@minecraft", "server")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	pkg := []byte(`{"name":"@minecraft/server","version":"2.7.0-beta.1.26.14-stable"}`)
	if err := os.WriteFile(filepath.Join(moduleDir, "package.json"), pkg, 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	warnings := validateInstalledDependencies(tmpDir, map[string]string{
		"@minecraft/server": "2.7.0-beta",
	})
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	warnings = validateInstalledDependencies(tmpDir, map[string]string{
		"@minecraft/server": "1.0.0",
	})
	if len(warnings) == 0 {
		t.Fatal("expected mismatch warning")
	}
}

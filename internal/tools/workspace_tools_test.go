package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectAndValidateWorkspace(t *testing.T) {
	root := t.TempDir()
	bp := filepath.Join(root, "behavior_pack")
	if err := os.MkdirAll(bp, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
  "format_version": 2,
  "header": {"name":"A","description":"B","uuid":"11111111-1111-1111-1111-111111111111","version":[1,0,0]},
  "modules": [{"type":"script","uuid":"22222222-2222-2222-2222-222222222222","version":[1,0,0],"language":"javascript","entry":"scripts/main.js"}],
  "dependencies": [{"module_name":"@minecraft/server","version":"2.0.0-beta"}]
}`
	if err := os.WriteFile(filepath.Join(bp, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(bp, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bp, "scripts", "main.js"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.js"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	ins, err := inspectAddonWorkspace(root)
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	if !ins.HasBehaviorPack || ins.Language != "javascript" {
		t.Fatalf("unexpected inspect output: %+v", ins)
	}

	val, err := validateAddonWorkspace(root)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if !val.Valid {
		t.Fatalf("expected valid workspace, findings: %+v", val.Findings)
	}
}

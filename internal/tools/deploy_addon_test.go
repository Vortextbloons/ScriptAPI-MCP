package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/mcdev"
)

func TestDeployAddon_ResolvesMCDevFromEnv(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "behavior_pack"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "behavior_pack", "manifest.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	comMojang := filepath.Join(t.TempDir(), "com.mojang")
	t.Setenv(mcdev.EnvPrimary, comMojang)

	out, err := deployAddon(DeployAddonInput{ProjectPath: project, DryRun: true})
	if err != nil {
		t.Fatalf("deployAddon failed: %v", err)
	}
	if out.MCDevPath != comMojang {
		t.Errorf("MCDevPath = %q, want %q", out.MCDevPath, comMojang)
	}
	if out.MCDevSource != "env" {
		t.Errorf("MCDevSource = %q, want env", out.MCDevSource)
	}
}

func TestDeployAddon_CreatesMissingDevDirs(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "behavior_pack"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "behavior_pack", "pack.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	comMojang := filepath.Join(t.TempDir(), "fresh", "com.mojang")

	out, err := deployAddon(DeployAddonInput{
		ProjectPath: project,
		MCDevPath:   comMojang,
		BPDestName:  "TestBP",
	})
	if err != nil {
		t.Fatalf("deployAddon failed: %v", err)
	}
	if out.MCDevSource != "explicit" {
		t.Errorf("MCDevSource = %q, want explicit", out.MCDevSource)
	}
	got, err := os.ReadFile(filepath.Join(comMojang, "development_behavior_packs", "TestBP", "pack.txt"))
	if err != nil {
		t.Fatalf("deployed file missing: %v", err)
	}
	if string(got) != "data" {
		t.Errorf("deployed content = %q", got)
	}
	if _, err := os.Stat(filepath.Join(comMojang, "development_resource_packs")); err != nil {
		t.Errorf("expected development_resource_packs to be created: %v", err)
	}
}

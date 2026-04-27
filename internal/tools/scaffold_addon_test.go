package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
)

func TestBuildScaffoldFiles(t *testing.T) {
	output := ScaffoldAddonOutput{
		BehaviorPackManifest: "bp-manifest",
		ResourcePackManifest: "rp-manifest",
		StarterCode: map[string]string{
			"src/main.js": "console.log('ok')",
		},
		PackageJSON:  "pkg-json",
		DeployScript: "deploy-js",
	}

	files := buildScaffoldFiles(output, true)

	if files[filepath.Join("behavior_pack", "manifest.json")] != "bp-manifest" {
		t.Fatalf("behavior pack manifest not included")
	}
	if files[filepath.Join("src", "main.js")] != "console.log('ok')" {
		t.Fatalf("starter code not included")
	}
	if files[filepath.Join("resource_pack", "manifest.json")] != "rp-manifest" {
		t.Fatalf("resource pack manifest not included")
	}
	if files["package.json"] != "pkg-json" {
		t.Fatalf("package.json not included")
	}
	if files[filepath.Join("scripts", "deploy.js")] != "deploy-js" {
		t.Fatalf("deploy script not included")
	}
}

func TestBuildVersionLookupSteps(t *testing.T) {
	steps := buildVersionLookupSteps([]string{"@minecraft/server", "@minecraft/server-ui"})

	if len(steps) != 2 {
		t.Fatalf("step count = %d, want 2", len(steps))
	}
	if steps[0].Command != "npm view @minecraft/server versions --json" {
		t.Fatalf("unexpected command for server: %q", steps[0].Command)
	}
	if steps[1].Command != "npm view @minecraft/server-ui versions --json" {
		t.Fatalf("unexpected command for server-ui: %q", steps[1].Command)
	}
}

func TestWriteScaffoldFiles(t *testing.T) {
	root := t.TempDir()
	written, err := writeScaffoldFiles(root, map[string]string{
		filepath.Join("behavior_pack", "manifest.json"): "bp-manifest",
		filepath.Join("src", "main.js"): "console.log('ok')",
		filepath.Join("scripts", "deploy.js"): "deploy-js",
	})
	if err != nil {
		t.Fatalf("writeScaffoldFiles returned error: %v", err)
	}

	if len(written) != 3 {
		t.Fatalf("written count = %d, want 3", len(written))
	}

	for _, rel := range []string{
		filepath.Join("behavior_pack", "manifest.json"),
		filepath.Join("src", "main.js"),
		filepath.Join("scripts", "deploy.js"),
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected file %s to exist: %v", rel, err)
		}
	}
}

func TestResolveScaffoldRoot(t *testing.T) {
	root, err := resolveScaffoldRoot("", "MyAddon", false)
	if err != nil {
		t.Fatalf("resolveScaffoldRoot returned error: %v", err)
	}
	if !strings.HasSuffix(root, "MyAddon") {
		t.Fatalf("root %q should end with addon name", root)
	}

	current, err := resolveScaffoldRoot("C:/tmp/project", "MyAddon", true)
	if err != nil {
		t.Fatalf("resolveScaffoldRoot returned error: %v", err)
	}
	if current != "C:/tmp/project" {
		t.Fatalf("current dir root = %q, want C:/tmp/project", current)
	}
}

func TestGenerateDeployScriptUsesPackFolders(t *testing.T) {
	script := generateDeployScript(ScaffoldAddonInput{
		AddonName:                     "MyAddon",
		MCDevPath:                     `C:\Users\isaac\AppData\Roaming\Minecraft Bedrock\Users\Shared\games\com.mojang`,
		BPFolderName:                  "behavior_pack",
		RPFolderName:                  "resource_pack",
		CreateDeployScript:            true,
		NeedsCustomBlocksItemsEntities: true,
	})

	if !strings.Contains(script, `path.join(__dirname, "..", "behavior_pack")`) {
		t.Fatalf("deploy script should reference behavior_pack source folder")
	}
	if !strings.Contains(script, `path.join(__dirname, "..", "resource_pack")`) {
		t.Fatalf("deploy script should reference resource_pack source folder")
	}
	if strings.Contains(script, `path.join(__dirname, "..", ".")`) {
		t.Fatalf("deploy script should not point at project root")
	}

	if !strings.Contains(script, `const mcDev = "C:\\Users\\isaac\\AppData\\Roaming\\Minecraft Bedrock\\Users\\Shared\\games\\com.mojang"`) {
		t.Fatalf("deploy script should embed the explicit mcdev path")
	}

	_ = manifest.GenerateUUID
}

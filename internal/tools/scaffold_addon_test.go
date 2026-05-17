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
	written, skipped, err := writeScaffoldFiles(root, map[string]string{
		filepath.Join("behavior_pack", "manifest.json"): "bp-manifest",
		filepath.Join("src", "main.js"):                 "console.log('ok')",
		filepath.Join("scripts", "deploy.js"):           "deploy-js",
	}, false)
	if err != nil {
		t.Fatalf("writeScaffoldFiles returned error: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("skipped count = %d, want 0", len(skipped))
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

func TestWriteScaffoldFilesNoOverwrite(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "src", "main.js")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed file failed: %v", err)
	}

	written, skipped, err := writeScaffoldFiles(root, map[string]string{
		filepath.Join("src", "main.js"): "new",
	}, false)
	if err != nil {
		t.Fatalf("writeScaffoldFiles returned error: %v", err)
	}
	if len(written) != 0 || len(skipped) != 1 {
		t.Fatalf("written=%d skipped=%d, want written=0 skipped=1", len(written), len(skipped))
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
		AddonName:                      "MyAddon",
		MCDevPath:                      `C:\Users\isaac\AppData\Roaming\Minecraft Bedrock\Users\Shared\games\com.mojang`,
		BPFolderName:                   "behavior_pack",
		RPFolderName:                   "resource_pack",
		CreateDeployScript:             true,
		NeedsCustomBlocksItemsEntities: true,
	})

	if !strings.Contains(script, `join(ROOT, "behavior_pack")`) {
		t.Fatalf("deploy script should reference behavior_pack source folder")
	}
	if !strings.Contains(script, `join(ROOT, "resource_pack")`) {
		t.Fatalf("deploy script should reference resource_pack source folder")
	}
	if !strings.Contains(script, `const ROOT = resolve(__dirname, "..")`) {
		t.Fatalf("deploy script should define ROOT from __dirname")
	}

	if !strings.Contains(script, `readEnv("DEPLOY_PATH")`) {
		t.Fatalf("deploy script should read DEPLOY_PATH from .env")
	}

	if !strings.Contains(script, `readEnv("DOWNLOAD_PATH")`) {
		t.Fatalf("deploy script should read DOWNLOAD_PATH from .env")
	}

	if !strings.Contains(script, `process.argv[2]`) {
		t.Fatalf("deploy script should route on process.argv[2]")
	}

	if !strings.Contains(script, `cmd === "dev"`) {
		t.Fatalf("deploy script should support dev command")
	}

	if !strings.Contains(script, `cmd === "prod"`) {
		t.Fatalf("deploy script should support prod command")
	}

	if !strings.Contains(script, `Compress-Archive`) {
		t.Fatalf("deploy script should include Compress-Archive for prod")
	}

	_ = manifest.GenerateUUID
}

func TestGenerateDeployScriptJSBuildStep(t *testing.T) {
	script := generateDeployScript(ScaffoldAddonInput{
		AddonName:         "MyAddon",
		ScriptingLanguage: "javascript",
	})

	if !strings.Contains(script, `Built src/main.js -> behavior_pack/scripts/main.js`) {
		t.Fatalf("JS deploy script should include build step that copies source")
	}

	if !strings.Contains(script, `existsSync(srcFile)`) {
		t.Fatalf("JS build step should check for src/main.js existence")
	}

	_ = manifest.GenerateUUID
}

func TestGenerateDeployScriptTSNoBuildStep(t *testing.T) {
	script := generateDeployScript(ScaffoldAddonInput{
		AddonName:         "MyAddon",
		ScriptingLanguage: "typescript",
	})

	if strings.Contains(script, `src/main.js`) {
		t.Fatalf("TS deploy script should not reference src/main.js")
	}

	if !strings.Contains(script, `handled by esbuild`) {
		t.Fatalf("TS deploy script build() should note esbuild handles build")
	}

	_ = manifest.GenerateUUID
}

func TestGenerateDeployScriptBPOnly(t *testing.T) {
	script := generateDeployScript(ScaffoldAddonInput{
		AddonName:                      "MyAddon",
		BPFolderName:                   "behavior_pack",
		NeedsCustomBlocksItemsEntities: false,
	})

	if !strings.Contains(script, `join(ROOT, "behavior_pack")`) {
		t.Fatalf("deploy script should reference behavior_pack")
	}

	if strings.Contains(script, `development_resource_packs`) {
		t.Fatalf("BP-only deploy script should not reference resource packs")
	}

	_ = manifest.GenerateUUID
}

func TestGenerateDeployScriptRPOnly(t *testing.T) {
	script := generateDeployScript(ScaffoldAddonInput{
		AddonName:                      "MyAddon",
		BPFolderName:                   "behavior_pack",
		RPFolderName:                   "resource_pack",
		NeedsCustomBlocksItemsEntities: true,
	})

	if !strings.Contains(script, `development_resource_packs`) {
		t.Fatalf("deploy script with RP should reference development_resource_packs")
	}

	if !strings.Contains(script, `ADDON_NAME + "_RP.mcpack"`) {
		t.Fatalf("prod command should create RP .mcpack")
	}

	_ = manifest.GenerateUUID
}

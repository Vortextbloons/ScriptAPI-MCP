package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/models"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

type ScaffoldAddonInput struct {
	AddonName                      string   `json:"addon_name" mcp:"required,description='Name of the add-on'"`
	AddonDescription               string   `json:"addon_description" mcp:"required,description='Short description of the add-on'"`
	NeedsCustomBlocksItemsEntities bool     `json:"needs_custom_blocks_items_entities" mcp:"description='Determines if a Resource Pack is needed'"`
	NeedsUIMenus                   bool     `json:"needs_ui_menus" mcp:"description='Determines if @minecraft/server-ui is added to dependencies'"`
	ScriptingLanguage              string   `json:"scripting_language" mcp:"required,description='javascript or typescript'"`
	ServerVersion                  string   `json:"server_version" mcp:"required,description='Target Minecraft game version (e.g., 1.21.70, 1.26.13). Used with dependency_channel to resolve actual npm versions'"`
	DependencyChannel              string   `json:"dependency_channel" mcp:"description='Release channel for dependencies: stable, beta, or preview (default: beta)'"`
	Dependencies                   []string `json:"dependencies" mcp:"description='List of @minecraft/* modules to include (e.g., [\"@minecraft/server\", \"@minecraft/server-ui\"])'"`
	CreateDeployScript             bool     `json:"create_deploy_script" mcp:"description='Whether to create a deploy.js script that copies build output to Minecraft com.mojang folder'"`
	WriteFiles                     bool     `json:"write_files" mcp:"description='Whether to write the generated scaffold to disk'"`
	MCDevPath                      string   `json:"mcdev_path" mcp:"required,description='Path to Minecraft Bedrock com.mojang folder'"`
	BPFolderName                   string   `json:"bp_folder_name" mcp:"description='Source behavior pack folder name in project (default: behavior_pack)'"`
	RPFolderName                   string   `json:"rp_folder_name" mcp:"description='Source resource pack folder name in project (default: resource_pack)'"`
	BPDestName                     string   `json:"bp_dest_name" mcp:"description='Destination folder name in development_behavior_packs (e.g., MyAddon or MyAddon BP)'"`
	RPDestName                     string   `json:"rp_dest_name" mcp:"description='Destination folder name in development_resource_packs (e.g., MyAddon RP or MyAddon Resources)'"`
	CreateInCurrentDir             bool     `json:"create_in_current_dir" mcp:"description='If true, creates addon in current directory instead of subfolder'"`
	ProjectPath                    string   `json:"project_path" mcp:"description='Project folder used for local node_modules validation'"`
	DryRun                         bool     `json:"dry_run" mcp:"description='If true, previews files and does not write to disk'"`
	OverwriteExisting              bool     `json:"overwrite_existing" mcp:"description='If true, allows overwriting existing files when write_files=true'"`
}

type ScaffoldAddonOutput struct {
	BehaviorPackManifest string              `json:"behavior_pack_manifest"`
	ResourcePackManifest string              `json:"resource_pack_manifest,omitempty"`
	FileStructure        []string            `json:"file_structure"`
	VersionLookupSteps   []VersionLookupStep `json:"version_lookup_steps,omitempty"`
	StarterCode          map[string]string   `json:"starter_code"`
	PackageJSON          string              `json:"package_json,omitempty"`
	BundleScript         string              `json:"bundle_script,omitempty"`
	DeployScript         string              `json:"deploy_script,omitempty"`
	WrittenFiles         []string            `json:"written_files,omitempty"`
	OutputPath           string              `json:"output_path,omitempty"`
	WouldWriteFiles      []string            `json:"would_write_files,omitempty"`
	SkippedFiles         []string            `json:"skipped_files,omitempty"`
	ValidationWarnings   []string            `json:"validation_warnings,omitempty"`
}

type VersionLookupStep struct {
	Module  string `json:"module"`
	Command string `json:"command"`
}

func RegisterScaffoldAddon(server *mcp.Server, npmClient *npm.Client) error {
	return server.RegisterTool("scaffold_addon",
		"Scaffolds a new Minecraft Bedrock addon with behavior pack, resource pack, and optional deploy script. First ask for the target Minecraft game version (e.g., 1.21.70, 1.26.13). Then ask what dependency channel to use (stable, beta, or preview) as this determines the correct dependency versions. Also ask the user to inspect npm versions with `npm view <module> versions --json` for each selected dependency before choosing the matching channel version. Set write_files=true to write the scaffold to disk. When create_deploy_script is true, also ask for the exact Minecraft com.mojang path plus BP/RP source and destination folder names.",
		func(args ScaffoldAddonInput) (*mcp.ToolResponse, error) {
			return handleScaffoldAddon(args, npmClient)
		})
}

func handleScaffoldAddon(args ScaffoldAddonInput, npmClient *npm.Client) (*mcp.ToolResponse, error) {
	if args.ScriptingLanguage != "javascript" && args.ScriptingLanguage != "typescript" {
		return toolErrorResponse("INVALID_INPUT", fmt.Sprintf("invalid scripting_language: %q", args.ScriptingLanguage), false, "Use javascript or typescript"), nil
	}
	if strings.TrimSpace(args.ServerVersion) == "" {
		return toolErrorResponse("INVALID_INPUT", "minecraft version is required", false, "Provide server_version like 1.21.70"), nil
	}

	// Validate channel
	channel := args.DependencyChannel
	if channel != "stable" && channel != "beta" && channel != "preview" {
		channel = "beta"
	}

	bpUUID := manifest.GenerateUUID()
	rpUUID := ""
	if args.NeedsCustomBlocksItemsEntities {
		rpUUID = manifest.GenerateUUID()
	}

	// Build dependencies
	var deps []models.Dependency
	var err error
	warnings := make([]string, 0)
	modules := []string{"@minecraft/server"}

	// If dependencies are explicitly provided, use them with channel resolution
	if len(args.Dependencies) > 0 {
		modules = append([]string(nil), args.Dependencies...)
		deps, err = manifest.BuildDependenciesWithChannel(npmClient, args.Dependencies, args.ServerVersion, channel)
		if err != nil {
			return toolErrorResponse("DEPENDENCY_RESOLVE_FAILED", fmt.Sprintf("error resolving dependencies: %v", err), false, "Check dependency names and channel"), nil
		}
		manifestVersions := make(map[string]string, len(deps))
		for _, dep := range deps {
			manifestVersions[dep.ModuleName] = dep.Version
		}
		warnings = append(warnings, validateInstalledDependencies(args.ProjectPath, manifestVersions)...)
	} else {
		// Fallback to legacy behavior with needs_ui_menus
		// Build a default module list based on needs_ui_menus
		if args.NeedsUIMenus {
			modules = append(modules, "@minecraft/server-ui")
		}
		deps, err = manifest.BuildDependenciesWithChannel(npmClient, modules, args.ServerVersion, channel)
		if err != nil {
			return toolErrorResponse("DEPENDENCY_RESOLVE_FAILED", fmt.Sprintf("error resolving dependencies: %v", err), false, "Check minecraft version and channel"), nil
		}
		manifestVersions := make(map[string]string, len(deps))
		for _, dep := range deps {
			manifestVersions[dep.ModuleName] = dep.Version
		}
		warnings = append(warnings, validateInstalledDependencies(args.ProjectPath, manifestVersions)...)
	}

	bp := manifest.GenerateBP(args.AddonName, args.AddonDescription, deps, bpUUID)
	bpJSON, err := manifest.FormatManifest(bp)
	if err != nil {
		return toolErrorResponse("MANIFEST_FORMAT_FAILED", fmt.Sprintf("error formatting BP manifest: %v", err), false), nil
	}

	output := ScaffoldAddonOutput{
		BehaviorPackManifest: bpJSON,
		FileStructure:        manifest.FileStructure(args.AddonName, args.NeedsCustomBlocksItemsEntities, args.ScriptingLanguage, args.CreateDeployScript),
		VersionLookupSteps:   buildVersionLookupSteps(modules),
		StarterCode:          manifest.GenerateStarterCode(args.ScriptingLanguage, args.ServerVersion),
		ValidationWarnings:   warnings,
	}

	if args.NeedsCustomBlocksItemsEntities {
		rp := manifest.GenerateRP(args.AddonName, args.AddonDescription, rpUUID, bpUUID)
		rpJSON, err := manifest.FormatManifest(rp)
		if err != nil {
			return toolErrorResponse("MANIFEST_FORMAT_FAILED", fmt.Sprintf("error formatting RP manifest: %v", err), false), nil
		}
		output.ResourcePackManifest = rpJSON
	}

	if args.CreateDeployScript {
		output.BundleScript = generateBundleScript(args)
		output.DeployScript = generateDeployScript(args)
		output.PackageJSON = manifest.GeneratePackageJSON(args.AddonName, deps, args.ScriptingLanguage)
		if runtime.GOOS != "windows" {
			output.ValidationWarnings = append(output.ValidationWarnings, "WARNING: generated deploy.js production packaging uses PowerShell Compress-Archive and is Windows-oriented.")
		}
	}

	if args.WriteFiles {
		rootPath, err := resolveScaffoldRoot(args.ProjectPath, args.AddonName, args.CreateInCurrentDir)
		if err != nil {
			return toolErrorResponse("OUTPUT_PATH_INVALID", fmt.Sprintf("error resolving output path: %v", err), false), nil
		}

		files := buildScaffoldFiles(output, args.CreateDeployScript)
		if args.DryRun {
			output.OutputPath = rootPath
			output.WouldWriteFiles = sortedPaths(files)
		} else {
			written, skipped, err := writeScaffoldFiles(rootPath, files, args.OverwriteExisting)
			if err != nil {
				return toolErrorResponse("WRITE_FAILED", fmt.Sprintf("error writing scaffold files: %v", err), false, "Set overwrite_existing=true to replace existing files", "Use dry_run=true to preview"), nil
			}

			output.OutputPath = rootPath
			output.WrittenFiles = written
			output.SkippedFiles = skipped
		}
	}

	jsonOut, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
}

func resolveScaffoldRoot(projectPath, addonName string, createInCurrentDir bool) (string, error) {
	basePath := strings.TrimSpace(projectPath)
	if basePath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		basePath = cwd
	}

	if createInCurrentDir {
		return basePath, nil
	}

	return filepath.Join(basePath, addonName), nil
}

func buildScaffoldFiles(output ScaffoldAddonOutput, includeDeploy bool) map[string]string {
	files := map[string]string{
		filepath.Join("behavior_pack", "manifest.json"): output.BehaviorPackManifest,
	}

	for relPath, content := range output.StarterCode {
		files[filepath.FromSlash(relPath)] = content
	}

	if output.ResourcePackManifest != "" {
		files[filepath.Join("resource_pack", "manifest.json")] = output.ResourcePackManifest
	}

	if includeDeploy {
		if output.PackageJSON != "" {
			files["package.json"] = output.PackageJSON
		}
		if output.BundleScript != "" {
			files[filepath.Join("scripts", "bundle.js")] = output.BundleScript
		}
		if output.DeployScript != "" {
			files[filepath.Join("scripts", "deploy.js")] = output.DeployScript
		}
	}

	return files
}

func sortedPaths(files map[string]string) []string {
	paths := make([]string, 0, len(files))
	for relPath := range files {
		paths = append(paths, relPath)
	}
	sort.Strings(paths)
	return paths
}

func writeScaffoldFiles(rootPath string, files map[string]string, overwrite bool) ([]string, []string, error) {
	if err := os.MkdirAll(rootPath, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create scaffold root: %w", err)
	}

	paths := sortedPaths(files)

	written := make([]string, 0, len(paths))
	skipped := make([]string, 0)
	for _, relPath := range paths {
		cleanRel := filepath.Clean(relPath)
		if filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, "..") {
			return nil, nil, fmt.Errorf("invalid relative path %q", relPath)
		}
		fullPath := filepath.Join(rootPath, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create directory for %s: %w", relPath, err)
		}
		if !overwrite {
			if _, err := os.Stat(fullPath); err == nil {
				skipped = append(skipped, relPath)
				continue
			}
		}
		if err := os.WriteFile(fullPath, []byte(files[relPath]), 0644); err != nil {
			return nil, nil, fmt.Errorf("failed to write %s: %w", relPath, err)
		}
		written = append(written, relPath)
	}

	return written, skipped, nil
}

func buildVersionLookupSteps(modules []string) []VersionLookupStep {
	steps := make([]VersionLookupStep, 0, len(modules))
	for _, module := range modules {
		steps = append(steps, VersionLookupStep{
			Module:  module,
			Command: fmt.Sprintf("npm view %s versions --json", module),
		})
	}
	return steps
}

func generateBundleScript(args ScaffoldAddonInput) string {
	bpFolder := "behavior_pack"
	rpFolder := "resource_pack"
	if args.BPFolderName != "" {
		bpFolder = args.BPFolderName
	}
	if args.RPFolderName != "" {
		rpFolder = args.RPFolderName
	}

	needsRP := args.NeedsCustomBlocksItemsEntities

	var script string
	script = `// Bundles the addon into a .mcaddon. Asks whether this is a dev pack first.
const { execSync } = require("child_process");
const { existsSync, readFileSync, writeFileSync } = require("fs");
const { resolve, join } = require("path");
const readline = require("readline");

const ROOT = resolve(__dirname, "..");
const BP_SRC = join(ROOT, "` + bpFolder + `");
const DEV_SUFFIX = "-dev";

function prompt(question) {
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
  return new Promise((resolveAnswer) => {
    rl.question(question, (answer) => {
      rl.close();
      resolveAnswer(answer.trim());
    });
  });
}

function patchManifest(manifestPath, isDev) {
  if (!existsSync(manifestPath)) return null;
  const original = readFileSync(manifestPath, "utf8");
  const manifest = JSON.parse(original);
  manifest.header = manifest.header || {};
  const name = manifest.header.name || "";
  let nextName = name;
  if (isDev && !name.endsWith(DEV_SUFFIX)) {
    nextName = name + DEV_SUFFIX;
  } else if (!isDev && name.endsWith(DEV_SUFFIX)) {
    nextName = name.slice(0, -DEV_SUFFIX.length);
  }
  if (nextName !== name) {
    manifest.header.name = nextName;
    writeFileSync(manifestPath, JSON.stringify(manifest, null, 2) + "\n");
  }
  return original;
}

function restoreManifest(manifestPath, original) {
  if (original != null) writeFileSync(manifestPath, original);
}

async function main() {
  const answer = await prompt("Is this a dev pack? (y/N): ");
  const isDev = /^y(es)?$/i.test(answer);

  const manifestPaths = [join(BP_SRC, "manifest.json")];
`
	if needsRP {
		script += `  manifestPaths.push(join(ROOT, "` + rpFolder + `", "manifest.json"));
`
	}
	script += `
  const patched = [];
  for (const manifestPath of manifestPaths) {
    const original = patchManifest(manifestPath, isDev);
    if (original != null) patched.push([manifestPath, original]);
  }

  try {
    execSync("npm run build", { stdio: "inherit", cwd: ROOT });
    const deployEnv = { ...process.env, SKIP_BUILD: "1" };
    if (isDev) deployEnv.DEV_PACK = "1";
    execSync("node scripts/deploy.js prod", { stdio: "inherit", cwd: ROOT, env: deployEnv });
  } finally {
    for (const [manifestPath, original] of patched) {
      restoreManifest(manifestPath, original);
    }
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
`
	return script
}

func generateDeployScript(args ScaffoldAddonInput) string {
	bpFolder := "behavior_pack"
	rpFolder := "resource_pack"
	if args.BPFolderName != "" {
		bpFolder = args.BPFolderName
	}
	if args.RPFolderName != "" {
		rpFolder = args.RPFolderName
	}

	bpDest := args.AddonName
	rpDest := args.AddonName + " Resources"
	if args.BPDestName != "" {
		bpDest = args.BPDestName
	}
	if args.RPDestName != "" {
		rpDest = args.RPDestName
	}

	needsRP := args.NeedsCustomBlocksItemsEntities
	lang := args.ScriptingLanguage
	addonName := args.AddonName
	needsJSBuild := lang != "typescript"

	var script string

	// --- header & constants ---
	script = `const { execSync } = require("child_process");
const { existsSync, cpSync, mkdirSync, rmSync, readFileSync, renameSync } = require("fs");
const { resolve, join } = require("path");

const ROOT = resolve(__dirname, "..");
const BP_SRC = join(ROOT, "` + bpFolder + `");
`
	if needsRP {
		script += `const RP_SRC = join(ROOT, "` + rpFolder + `");
`
	}
	script += `const BP_DEST_NAME = "` + bpDest + `";
`
	if needsRP {
		script += `const RP_DEST_NAME = "` + rpDest + `";
`
	}
	script += `const ADDON_NAME = "` + addonName + `";

function readEnv(key) {
  const envPath = join(ROOT, ".env");
  if (!existsSync(envPath)) return "";
  const env = readFileSync(envPath, "utf8");
  for (const line of env.split("\n")) {
    const trimmed = line.trim();
    if (trimmed && !trimmed.startsWith("#")) {
      const eq = trimmed.indexOf("=");
      if (eq > 0 && trimmed.slice(0, eq).trim() === key) {
        return trimmed.slice(eq + 1).trim();
      }
    }
  }
  return "";
}

`
	// --- build step ---
	if needsJSBuild {
		script += `function build() {
  const srcFile = join(ROOT, "src", "main.js");
  const outDir = join(ROOT, "behavior_pack", "scripts");
  const outFile = join(outDir, "main.js");
  if (existsSync(srcFile)) {
    if (!existsSync(outDir)) mkdirSync(outDir, { recursive: true });
    cpSync(srcFile, outFile);
    console.log("  Built src/main.js -> behavior_pack/scripts/main.js");
  }
}

`
	} else {
		script += `function build() {
  // TypeScript: build handled by esbuild via package.json
}

`
	}

	// --- dev command ---
	script += `function dev() {
  if (!process.env.SKIP_BUILD) build();
  const mcDev = readEnv("DEPLOY_PATH");
  if (!mcDev) {
    console.error("DEPLOY_PATH not set in .env");
    process.exit(1);
  }
  const bDest = join(mcDev, "development_behavior_packs", BP_DEST_NAME);
  rmSync(bDest, { recursive: true, force: true });
  mkdirSync(bDest, { recursive: true });
  cpSync(BP_SRC, bDest, { recursive: true, force: true });
  console.log("Deployed BP: " + bDest);
`
	if needsRP {
		script += `  const rDest = join(mcDev, "development_resource_packs", RP_DEST_NAME);
  rmSync(rDest, { recursive: true, force: true });
  mkdirSync(rDest, { recursive: true });
  cpSync(RP_SRC, rDest, { recursive: true, force: true });
  console.log("Deployed RP: " + rDest);
`
	}
	script += `}

`
	// --- prod command ---
	script += `function prod() {
  if (!process.env.SKIP_BUILD) build();
  const outPath = readEnv("DOWNLOAD_PATH");
  if (!outPath) {
    console.error("DOWNLOAD_PATH not set in .env");
    process.exit(1);
  }
  const tempDir = join(ROOT, "temp_release");
  if (existsSync(tempDir)) rmSync(tempDir, { recursive: true });
  mkdirSync(tempDir, { recursive: true });

  console.log("Zipping behavior pack...");
  const bpMcpack = join(tempDir, ADDON_NAME + "_BP.mcpack");
  execSync(` + "`" + `powershell -NoProfile -Command "Compress-Archive -Path '${BP_SRC}\\*' -DestinationPath '${bpMcpack}' -Force"` + "`" + `, { stdio: "pipe" });
  const packs = ["'" + bpMcpack + "'"];
`
	if needsRP {
		script += `  console.log("Zipping resource pack...");
  const rpMcpack = join(tempDir, ADDON_NAME + "_RP.mcpack");
  execSync(` + "`" + `powershell -NoProfile -Command "Compress-Archive -Path '${RP_SRC}\\*' -DestinationPath '${rpMcpack}' -Force"` + "`" + `, { stdio: "pipe" });
  packs.push("'" + rpMcpack + "'");
`
	}
	script += `  console.log("Creating .mcaddon...");
  const releaseName = process.env.DEV_PACK === "1" ? ADDON_NAME + "-dev" : ADDON_NAME;
  const addonZip = join(tempDir, releaseName + ".zip");
  execSync(` + "`" + `powershell -NoProfile -Command "Compress-Archive -Path ${packs.join(',')} -DestinationPath '${addonZip}' -Force"` + "`" + `, { stdio: "pipe" });

  const outputPath = join(outPath, releaseName + ".mcaddon");
  renameSync(addonZip, outputPath);
  rmSync(tempDir, { recursive: true });
  console.log("Created " + outputPath);
}

const cmd = process.argv[2];
if (cmd === "dev") dev();
else if (cmd === "prod") prod();
else if (cmd === "compile") build();
else {
  console.log("Usage: node scripts/deploy.js <dev|prod|compile>");
  console.log("  dev     - Build and deploy to Minecraft development folders");
  console.log("  prod    - Build and create .mcaddon for distribution");
  console.log("  compile - Copy/compile scripts only (used by npm run build)");
  console.log("");
  console.log("To package interactively, run: npm run bundle");
  process.exit(1);
}
`
	return script
}


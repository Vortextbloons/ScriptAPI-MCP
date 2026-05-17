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
  build();
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
  build();
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
  const addonZip = join(tempDir, ADDON_NAME + ".zip");
  execSync(` + "`" + `powershell -NoProfile -Command "Compress-Archive -Path ${packs.join(',')} -DestinationPath '${addonZip}' -Force"` + "`" + `, { stdio: "pipe" });

  const outputPath = join(outPath, ADDON_NAME + ".mcaddon");
  renameSync(addonZip, outputPath);
  rmSync(tempDir, { recursive: true });
  console.log("Created " + outputPath);
}

const cmd = process.argv[2];
if (cmd === "dev") dev();
else if (cmd === "prod") prod();
else {
  console.log("Usage: node scripts/deploy.js <dev|prod>");
  console.log("  dev   - Build and deploy to Minecraft development folders");
  console.log("  prod  - Build and create .mcaddon for distribution");
  process.exit(1);
}
`
	return script
}

func RunScaffoldAddonCLI() error {
	fmt.Println("=== Minecraft Bedrock Addon Scaffolding ===")
	fmt.Println()

	var addonName, addonDescription, serverVersion string
	var needsRP, needsUI bool
	var lang string
	var channel string
	var depsInput string

	fmt.Print("Add-on name: ")
	fmt.Scanln(&addonName)
	if addonName == "" {
		return fmt.Errorf("addon name is required")
	}

	fmt.Print("Add-on description: ")
	fmt.Scanln(&addonDescription)

	fmt.Print("Scripting language (javascript/typescript) [javascript]: ")
	fmt.Scanln(&lang)
	if lang == "" {
		lang = "javascript"
	}

	fmt.Print("Target Minecraft game version (e.g., 1.21.70, 1.26.13): ")
	fmt.Scanln(&serverVersion)
	if serverVersion == "" {
		return fmt.Errorf("minecraft version is required")
	}

	fmt.Print("Dependency channel (stable/beta) [beta]: ")
	fmt.Scanln(&channel)
	if channel == "" {
		channel = "beta"
	}

	fmt.Println()
	fmt.Println("Available modules:")
	fmt.Println("  - @minecraft/server")
	fmt.Println("  - @minecraft/server-ui")
	fmt.Println("  - @minecraft/server-net")
	fmt.Println("  - @minecraft/server-admin")
	fmt.Println("  - @minecraft/server-gametest")
	fmt.Println()
	fmt.Print("Enter dependencies (comma-separated, or press Enter for @minecraft/server only): ")
	fmt.Scanln(&depsInput)

	var dependencies []string
	if depsInput != "" {
		for _, dep := range splitAndTrim(depsInput, ",") {
			if dep != "" {
				dependencies = append(dependencies, dep)
			}
		}
	}

	if len(dependencies) == 0 {
		dependencies = []string{"@minecraft/server"}
	}

	fmt.Print("Needs Resource Pack (custom blocks/items/entities)? (y/N): ")
	var rpInput string
	fmt.Scanln(&rpInput)
	needsRP = rpInput == "y" || rpInput == "Y"

	fmt.Print("Needs UI menus (@minecraft/server-ui)? (y/N): ")
	var uiInput string
	fmt.Scanln(&uiInput)
	needsUI = uiInput == "y" || uiInput == "Y"

	if needsUI {
		hasUI := false
		for _, dep := range dependencies {
			if dep == "@minecraft/server-ui" {
				hasUI = true
				break
			}
		}
		if !hasUI {
			dependencies = append(dependencies, "@minecraft/server-ui")
		}
	}

	fmt.Print("Create deploy script? (y/N): ")
	var deployInput string
	fmt.Scanln(&deployInput)
	createDeploy := deployInput == "y" || deployInput == "Y"

	fmt.Print("Write files to disk? (y/N): ")
	var writeInput string
	fmt.Scanln(&writeInput)
	writeFiles := writeInput == "y" || writeInput == "Y"

	var mcdevPath, bpFolder, rpFolder, bpDest, rpDest string

	if createDeploy {
		fmt.Println("\n--- Deploy Script Configuration ---")
		fmt.Print("Minecraft com.mojang path: ")
		fmt.Scanln(&mcdevPath)
		if mcdevPath == "" {
			return fmt.Errorf("minecraft com.mojang path is required")
		}

		fmt.Print("Behavior pack source folder name [behavior_pack]: ")
		fmt.Scanln(&bpFolder)

		fmt.Print("Resource pack source folder name [resource_pack]: ")
		fmt.Scanln(&rpFolder)

		fmt.Printf("Deployed BP folder name [%s BP]: ", addonName)
		fmt.Scanln(&bpDest)

		fmt.Printf("Deployed RP folder name [%s RP]: ", addonName)
		fmt.Scanln(&rpDest)
	}

	input := ScaffoldAddonInput{
		AddonName:                      addonName,
		AddonDescription:               addonDescription,
		NeedsCustomBlocksItemsEntities: needsRP,
		NeedsUIMenus:                   needsUI,
		ScriptingLanguage:              lang,
		ServerVersion:                  serverVersion,
		DependencyChannel:              channel,
		Dependencies:                   dependencies,
		CreateDeployScript:             createDeploy,
		WriteFiles:                     writeFiles,
		MCDevPath:                      mcdevPath,
		BPFolderName:                   bpFolder,
		RPFolderName:                   rpFolder,
		BPDestName:                     bpDest,
		RPDestName:                     rpDest,
	}

	output, err := handleScaffoldAddon(input, npm.NewClient())
	if err != nil {
		return err
	}

	fmt.Println("\n=== Generated Output ===")
	jsonOut, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonOut))

	return nil
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range split(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func split(s, sep string) []string {
	result := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func EnsureAddonDirectories(basePath string, addonName string, needsRP bool, lang string) error {
	dirs := []string{
		filepath.Join(basePath, addonName, "behavior_pack", "scripts"),
		filepath.Join(basePath, addonName, "behavior_pack", "ui"),
	}

	if needsRP {
		dirs = append(dirs,
			filepath.Join(basePath, addonName, "resource_pack", "textures"),
			filepath.Join(basePath, addonName, "resource_pack", "ui"),
		)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func WriteAddonFiles(basePath string, addonName string, needsRP bool, lang string, bpManifest, rpManifest string, starterCode map[string]string) error {
	bpPath := filepath.Join(basePath, addonName, "behavior_pack")
	rpPath := filepath.Join(basePath, addonName, "resource_pack")

	if err := os.WriteFile(filepath.Join(bpPath, "manifest.json"), []byte(bpManifest), 0644); err != nil {
		return fmt.Errorf("failed to write BP manifest: %w", err)
	}

	filenames := make([]string, 0, len(starterCode))
	for filename := range starterCode {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		content := starterCode[filename]
		fullPath := filepath.Join(bpPath, filename)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", filename, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	if needsRP {
		if err := os.WriteFile(filepath.Join(rpPath, "manifest.json"), []byte(rpManifest), 0644); err != nil {
			return fmt.Errorf("failed to write RP manifest: %w", err)
		}
	}

	return nil
}

func CmdScaffoldAddon(args []string) error {
	if len(args) < 5 {
		fmt.Println("Usage: scaffold_addon <name> <description> <language> <server_version> [options]")
		fmt.Println("Options:")
		fmt.Println("  --needs-rp          Include resource pack")
		fmt.Println("  --needs-ui          Include server-ui dependency")
		fmt.Println("  --create-deploy     Generate deploy script")
		fmt.Println("  --mcdev <path>      Minecraft dev path")
		fmt.Println("  --bp-folder <name>  BP folder name")
		fmt.Println("  --rp-folder <name>  RP folder name")
		fmt.Println("  --bp-dest <name>    Deployed BP folder name")
		fmt.Println("  --rp-dest <name>    Deployed RP folder name")
		fmt.Println("  --project-path <path> Validation root for installed node_modules")
		fmt.Println("  --channel <stable|beta>  Dependency release channel (default: beta)")
		fmt.Println("  --deps <mod1,mod2>  Comma-separated list of @minecraft/* modules")
		return nil
	}

	input := ScaffoldAddonInput{
		AddonName:         args[0],
		AddonDescription:  args[1],
		ScriptingLanguage: args[2],
		ServerVersion:     args[3],
	}

	i := 4
	for i < len(args) {
		switch args[i] {
		case "--needs-rp":
			input.NeedsCustomBlocksItemsEntities = true
		case "--needs-ui":
			input.NeedsUIMenus = true
		case "--create-deploy":
			input.CreateDeployScript = true
		case "--write-files":
			input.WriteFiles = true
		case "--mcdev":
			if i+1 < len(args) {
				input.MCDevPath = args[i+1]
				i++
			}
		case "--bp-folder":
			if i+1 < len(args) {
				input.BPFolderName = args[i+1]
				i++
			}
		case "--rp-folder":
			if i+1 < len(args) {
				input.RPFolderName = args[i+1]
				i++
			}
		case "--bp-dest":
			if i+1 < len(args) {
				input.BPDestName = args[i+1]
				i++
			}
		case "--rp-dest":
			if i+1 < len(args) {
				input.RPDestName = args[i+1]
				i++
			}
		case "--channel":
			if i+1 < len(args) {
				input.DependencyChannel = args[i+1]
				i++
			}
		case "--deps":
			if i+1 < len(args) {
				input.Dependencies = splitAndTrim(args[i+1], ",")
				i++
			}
		case "--project-path":
			if i+1 < len(args) {
				input.ProjectPath = args[i+1]
				i++
			}
		}
		i++
	}

	output, err := handleScaffoldAddon(input, npm.NewClient())
	if err != nil {
		return err
	}

	jsonOut, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonOut))

	return nil
}

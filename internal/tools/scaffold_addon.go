package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
)

type ScaffoldAddonInput struct {
	AddonName                      string `json:"addon_name" mcp:"required,description='Name of the add-on'"`
	AddonDescription               string `json:"addon_description" mcp:"required,description='Short description of the add-on'"`
	NeedsCustomBlocksItemsEntities bool   `json:"needs_custom_blocks_items_entities" mcp:"description='Determines if a Resource Pack is needed'"`
	NeedsUIMenus                   bool   `json:"needs_ui_menus" mcp:"description='Determines if @minecraft/server-ui is added to dependencies'"`
	ScriptingLanguage              string `json:"scripting_language" mcp:"required,description='javascript or typescript'"`
	ServerVersion                  string `json:"server_version" mcp:"required,description='Resolved npm version from Tool 1'"`
	CreateDeployScript             bool   `json:"create_deploy_script" mcp:"description='Whether to create a build/deploy script'"`
	MCDevPath                      string `json:"mcdev_path" mcp:"description='Minecraft development path (default: C:/Users/<username>/AppData/Roaming/Minecraft Bedrock/Users/Shared/games/com.mojang)'"`
	BPFolderName                   string `json:"bp_folder_name" mcp:"description='Behavior pack folder name in project (default: behavior_pack)'"`
	RPFolderName                    string `json:"rp_folder_name" mcp:"description='Resource pack folder name in project (default: resource_pack)'"`
	BPDestName                     string `json:"bp_dest_name" mcp:"description='Deployed BP folder name (e.g., MyAddon or MyAddon BP)'"`
	RPDestName                      string `json:"rp_dest_name" mcp:"description='Deployed RP folder name (e.g., MyAddon RP or MyAddon Resources)'"`
	CreateInCurrentDir             bool   `json:"create_in_current_dir" mcp:"description='If true, creates addon in current directory instead of subfolder'"`
}

type ScaffoldAddonOutput struct {
	BehaviorPackManifest string            `json:"behavior_pack_manifest"`
	ResourcePackManifest string            `json:"resource_pack_manifest,omitempty"`
	FileStructure        []string          `json:"file_structure"`
	StarterCode          map[string]string `json:"starter_code"`
	DeployScript         string            `json:"deploy_script,omitempty"`
}

func RegisterScaffoldAddon(server *mcp.Server) error {
	return server.RegisterTool("scaffold_addon",
		"Scaffolds a new Minecraft Bedrock addon with behavior pack, resource pack, and optional deploy script. Asks about deploy script if not specified.",
		func(args ScaffoldAddonInput) (*mcp.ToolResponse, error) {
			return handleScaffoldAddon(args)
		})
}

func handleScaffoldAddon(args ScaffoldAddonInput) (*mcp.ToolResponse, error) {
	if args.ScriptingLanguage != "javascript" && args.ScriptingLanguage != "typescript" {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Invalid scripting_language: %q. Must be 'javascript' or 'typescript'.", args.ScriptingLanguage))), nil
	}

	bpUUID := manifest.GenerateUUID()
	rpUUID := ""
	if args.NeedsCustomBlocksItemsEntities {
		rpUUID = manifest.GenerateUUID()
	}

	deps := manifest.BuildDependencies(args.ServerVersion, args.NeedsUIMenus)
	bp := manifest.GenerateBP(args.AddonName, args.AddonDescription, deps, bpUUID)
	bpJSON, err := manifest.FormatManifest(bp)
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error formatting BP manifest: %v", err))), nil
	}

	output := ScaffoldAddonOutput{
		BehaviorPackManifest: bpJSON,
		FileStructure:        manifest.FileStructure(args.AddonName, args.NeedsCustomBlocksItemsEntities, args.ScriptingLanguage),
		StarterCode:          manifest.GenerateStarterCode(args.ScriptingLanguage, args.ServerVersion),
	}

	if args.NeedsCustomBlocksItemsEntities {
		rp := manifest.GenerateRP(args.AddonName, args.AddonDescription, rpUUID, bpUUID)
		rpJSON, err := manifest.FormatManifest(rp)
		if err != nil {
			return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error formatting RP manifest: %v", err))), nil
		}
		output.ResourcePackManifest = rpJSON
	}

	if args.CreateDeployScript {
		output.DeployScript = generateDeployScript(args)
	}

	jsonOut, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
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

	bpDest := args.AddonName + " BP"
	rpDest := args.AddonName + " RP"
	if args.BPDestName != "" {
		bpDest = args.BPDestName
	}
	if args.RPDestName != "" {
		rpDest = args.RPDestName
	}

	devPath := getDefaultMCDevPath()
	if args.MCDevPath != "" {
		devPath = args.MCDevPath
	}

	script := fmt.Sprintf(`const { cpSync, mkdirSync, rmSync } = require("fs");
const { resolve } = require("path");

const mcDev = "%s";
const bpSrc = resolve(__dirname, "..", "%s");
const rpSrc = resolve(__dirname, "..", "%s");
const bpDest = mcDev + "/development_behavior_packs/%s";
const rpDest = mcDev + "/development_resource_packs/%s";

for (const [src, dest, name] of [[bpSrc, bpDest, "BP"], [rpSrc, rpDest, "RP"]]) {
  rmSync(dest, { recursive: true, force: true });
  mkdirSync(dest, { recursive: true });
  cpSync(src, dest, { recursive: true, force: true });
  console.log("Deployed " + name + ": " + dest);
}
`, devPath, bpFolder, rpFolder, bpDest, rpDest)

	return script
}

func getDefaultMCDevPath() string {
	username := os.Getenv("USERNAME")
	if username == "" {
		username = "User"
	}
	return fmt.Sprintf("C:/Users/%s/AppData/Roaming/Minecraft Bedrock/Users/Shared/games/com.mojang", username)
}

func RunScaffoldAddonCLI() error {
	fmt.Println("=== Minecraft Bedrock Addon Scaffolding ===")
	fmt.Println()

	var addonName, addonDescription, serverVersion string
	var needsRP, needsUI bool
	var lang string

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

	fmt.Print("Server version (e.g., 1.0.0-beta.1.20.80-preview.24): ")
	fmt.Scanln(&serverVersion)

	fmt.Print("Needs Resource Pack (custom blocks/items/entities)? (y/N): ")
	var rpInput string
	fmt.Scanln(&rpInput)
	needsRP = rpInput == "y" || rpInput == "Y"

	fmt.Print("Needs UI menus (@minecraft/server-ui)? (y/N): ")
	var uiInput string
	fmt.Scanln(&uiInput)
	needsUI = uiInput == "y" || uiInput == "Y"

	fmt.Print("Create deploy script? (y/N): ")
	var deployInput string
	fmt.Scanln(&deployInput)
	createDeploy := deployInput == "y" || deployInput == "Y"

	input := ScaffoldAddonInput{
		AddonName:                      addonName,
		AddonDescription:               addonDescription,
		NeedsCustomBlocksItemsEntities: needsRP,
		NeedsUIMenus:                   needsUI,
		ScriptingLanguage:              lang,
		ServerVersion:                  serverVersion,
		CreateDeployScript:             createDeploy,
	}

	output, err := handleScaffoldAddon(input)
	if err != nil {
		return err
	}

	fmt.Println("\n=== Generated Output ===")
	jsonOut, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonOut))

	return nil
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

	for filename, content := range starterCode {
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
		}
		i++
	}

	output, err := handleScaffoldAddon(input)
	if err != nil {
		return err
	}

	jsonOut, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(jsonOut))

	return nil
}
package tools

import (
	"encoding/json"
	"fmt"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/models"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

// InitAddonWorkspaceInput is the input schema for Tool 2
type InitAddonWorkspaceInput struct {
	AddonName                      string   `json:"addon_name" mcp:"required,description='Name of the add-on'"`
	AddonDescription               string   `json:"addon_description" mcp:"required,description='Short description of the add-on'"`
	NeedsCustomBlocksItemsEntities bool     `json:"needs_custom_blocks_items_entities" mcp:"description='Determines if a Resource Pack is needed'"`
	NeedsUIMenus                   bool     `json:"needs_ui_menus" mcp:"description='Determines if @minecraft/server-ui is added to dependencies'"`
	ScriptingLanguage              string   `json:"scripting_language" mcp:"required,description='javascript or typescript'"`
	ServerVersion                  string   `json:"server_version" mcp:"required,description='Target Minecraft game version (e.g., 1.21.70, 1.26.13). Used with dependency_channel to resolve actual npm versions'"`
	DependencyChannel              string   `json:"dependency_channel" mcp:"description='Release channel for dependencies: stable, beta, or preview (default: beta)'"`
	Dependencies                   []string `json:"dependencies" mcp:"description='List of @minecraft/* modules to include (e.g., [\"@minecraft/server\", \"@minecraft/server-ui\"])'"`
	ProjectPath                    string   `json:"project_path" mcp:"description='Project folder used for local node_modules validation'"`
}

// InitAddonWorkspaceOutput is the output schema for Tool 2
type InitAddonWorkspaceOutput struct {
	BehaviorPackManifest string            `json:"behavior_pack_manifest"`
	ResourcePackManifest string            `json:"resource_pack_manifest,omitempty"`
	FileStructure        []string          `json:"file_structure"`
	StarterCode          map[string]string `json:"starter_code"`
	ValidationWarnings   []string          `json:"validation_warnings,omitempty"`
}

// RegisterInitAddonWorkspace registers Tool 2
func RegisterInitAddonWorkspace(server *mcp.Server) error {
	return server.RegisterTool("init_addon_workspace",
		"Generates folder structure, valid manifests with v4 UUIDs, and starter code. First ask for the target Minecraft game version (e.g., 1.21.70, 1.26.13). Then ask what dependency channel to use (stable, beta, or preview) as this determines the correct dependency versions.",
		func(args InitAddonWorkspaceInput) (*mcp.ToolResponse, error) {
			return handleInitAddonWorkspace(args)
		})
}

func handleInitAddonWorkspace(args InitAddonWorkspaceInput) (*mcp.ToolResponse, error) {
	// Validate language
	lang := args.ScriptingLanguage
	if lang != "javascript" && lang != "typescript" {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Invalid scripting_language: %q. Must be 'javascript' or 'typescript'.", lang))), nil
	}

	// Validate channel
	channel := args.DependencyChannel
	if channel != "stable" && channel != "beta" && channel != "preview" {
		channel = "beta"
	}

	// Generate UUIDs
	bpUUID := manifest.GenerateUUID()
	rpUUID := ""
	if args.NeedsCustomBlocksItemsEntities {
		rpUUID = manifest.GenerateUUID()
	}

	// Build dependencies
	var deps []models.Dependency
	var err error
	warnings := make([]string, 0)

	// If dependencies are explicitly provided, use them with channel resolution
	if len(args.Dependencies) > 0 {
		npmClient := npm.NewClient()
		deps, err = manifest.BuildDependenciesWithChannel(npmClient, args.Dependencies, args.ServerVersion, channel)
		if err != nil {
			return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error resolving dependencies: %v", err))), nil
		}
		manifestVersions := make(map[string]string, len(deps))
		for _, dep := range deps {
			manifestVersions[dep.ModuleName] = dep.Version
		}
		warnings = append(warnings, validateInstalledDependencies(args.ProjectPath, manifestVersions)...)
	} else {
		// Fallback to legacy behavior with needs_ui_menus
		// Build a default module list based on needs_ui_menus
		modules := []string{"@minecraft/server"}
		if args.NeedsUIMenus {
			modules = append(modules, "@minecraft/server-ui")
		}
		npmClient := npm.NewClient()
		deps, err = manifest.BuildDependenciesWithChannel(npmClient, modules, args.ServerVersion, channel)
		if err != nil {
			return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error resolving dependencies: %v", err))), nil
		}
		manifestVersions := make(map[string]string, len(deps))
		for _, dep := range deps {
			manifestVersions[dep.ModuleName] = dep.Version
		}
		warnings = append(warnings, validateInstalledDependencies(args.ProjectPath, manifestVersions)...)
	}

	// Create manifests
	bp := manifest.GenerateBP(args.AddonName, args.AddonDescription, deps, bpUUID)
	bpJSON, err := manifest.FormatManifest(bp)
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error formatting BP manifest: %v", err))), nil
	}

	output := InitAddonWorkspaceOutput{
		BehaviorPackManifest: bpJSON,
		FileStructure:        manifest.FileStructure(args.AddonName, args.NeedsCustomBlocksItemsEntities, lang, false),
		StarterCode:          manifest.GenerateStarterCode(lang, args.ServerVersion),
		ValidationWarnings:   warnings,
	}

	if args.NeedsCustomBlocksItemsEntities {
		rp := manifest.GenerateRP(args.AddonName, args.AddonDescription, rpUUID, bpUUID)
		rpJSON, err := manifest.FormatManifest(rp)
		if err != nil {
			return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error formatting RP manifest: %v", err))), nil
		}
		output.ResourcePackManifest = rpJSON
	}

	jsonOut, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
}

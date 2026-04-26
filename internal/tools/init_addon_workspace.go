package tools

import (
	"encoding/json"
	"fmt"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
)

// InitAddonWorkspaceInput is the input schema for Tool 2
type InitAddonWorkspaceInput struct {
	AddonName                      string `json:"addon_name" mcp:"required,description='Name of the add-on'"`
	AddonDescription               string `json:"addon_description" mcp:"required,description='Short description of the add-on'"`
	NeedsCustomBlocksItemsEntities bool   `json:"needs_custom_blocks_items_entities" mcp:"description='Determines if a Resource Pack is needed'"`
	NeedsUIMenus                   bool   `json:"needs_ui_menus" mcp:"description='Determines if @minecraft/server-ui is added to dependencies'"`
	ScriptingLanguage              string `json:"scripting_language" mcp:"required,description='javascript or typescript'"`
	ServerVersion                  string `json:"server_version" mcp:"required,description='Resolved npm version from Tool 1'"`
}

// InitAddonWorkspaceOutput is the output schema for Tool 2
type InitAddonWorkspaceOutput struct {
	BehaviorPackManifest string            `json:"behavior_pack_manifest"`
	ResourcePackManifest string            `json:"resource_pack_manifest,omitempty"`
	FileStructure        []string          `json:"file_structure"`
	StarterCode          map[string]string `json:"starter_code"`
}

// RegisterInitAddonWorkspace registers Tool 2
func RegisterInitAddonWorkspace(server *mcp.Server) error {
	return server.RegisterTool("init_addon_workspace",
		"Generates folder structure, valid manifests with v4 UUIDs, and starter code. Reduces setup to 5 critical questions.",
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

	// Generate UUIDs
	bpUUID := manifest.GenerateUUID()
	rpUUID := ""
	if args.NeedsCustomBlocksItemsEntities {
		rpUUID = manifest.GenerateUUID()
	}

	// Build dependencies
	deps := manifest.BuildDependencies(args.ServerVersion, args.NeedsUIMenus)

	// Create manifests
	bp := manifest.GenerateBP(args.AddonName, args.AddonDescription, deps, bpUUID)
	bpJSON, err := manifest.FormatManifest(bp)
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error formatting BP manifest: %v", err))), nil
	}

	output := InitAddonWorkspaceOutput{
		BehaviorPackManifest: bpJSON,
		FileStructure:        manifest.FileStructure(args.AddonName, args.NeedsCustomBlocksItemsEntities, lang),
		StarterCode:          manifest.GenerateStarterCode(lang, args.ServerVersion),
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

package tools

import (
	"fmt"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
)

// SyncManifestDepsInput is the input schema for Tool 4
type SyncManifestDepsInput struct {
	CurrentManifestJSON string   `json:"current_manifest_json" mcp:"required,description='The existing manifest.json as a string'"`
	AddedModules        []string `json:"added_modules" mcp:"description='Modules to add (e.g. @minecraft/server-net)'"`
	RemovedModules      []string `json:"removed_modules" mcp:"description='Modules to remove'"`
}

// RegisterSyncManifestDependencies registers Tool 4
func RegisterSyncManifestDependencies(server *mcp.Server) error {
	return server.RegisterTool("sync_manifest_dependencies",
		"Safely updates an existing manifest.json when adding or removing script modules. Fails explicitly if deprecated modules are passed.",
		func(args SyncManifestDepsInput) (*mcp.ToolResponse, error) {
			return handleSyncManifestDependencies(args)
		})
}

func handleSyncManifestDependencies(args SyncManifestDepsInput) (*mcp.ToolResponse, error) {
	// Parse existing manifest
	m, err := manifest.ParseManifest(args.CurrentManifestJSON)
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error parsing manifest: %v", err))), nil
	}

	// Apply changes
	if err := manifest.UpdateDependencies(&m, args.AddedModules, args.RemovedModules); err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Validation error: %v", err))), nil
	}

	// Format back to JSON
	updated, err := manifest.FormatManifest(m)
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error formatting manifest: %v", err))), nil
	}

	return mcp.NewToolResponse(mcp.NewTextContent(updated)), nil
}

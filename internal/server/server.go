package server

import (
	"fmt"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport"
	"github.com/metoro-io/mcp-golang/transport/stdio"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/resources"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/tools"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/version"
)

// New creates and configures the MCP server with all tools and resources using stdio transport.
func New() (*mcp.Server, error) {
	return NewWithTransport(stdio.NewStdioServerTransport())
}

// NewWithTransport creates and configures the MCP server with a custom transport.
func NewWithTransport(tr transport.Transport) (*mcp.Server, error) {
	server := mcp.NewServer(tr,
		mcp.WithName(version.Name),
		mcp.WithVersion(version.Current),
	)

	// Initialize npm client
	npmClient := npm.NewClient()

	// Register all tools
	if err := tools.RegisterResolveAPIEnvironment(server, npmClient); err != nil {
		return nil, fmt.Errorf("failed to register resolve_api_environment: %w", err)
	}
	if err := tools.RegisterInitAddonWorkspace(server); err != nil {
		return nil, fmt.Errorf("failed to register init_addon_workspace: %w", err)
	}
	if err := tools.RegisterSearchAPITypes(server, npmClient); err != nil {
		return nil, fmt.Errorf("failed to register search_api_types: %w", err)
	}
	if err := tools.RegisterSyncManifestDependencies(server); err != nil {
		return nil, fmt.Errorf("failed to register sync_manifest_dependencies: %w", err)
	}
	if err := tools.RegisterVersionInfo(server); err != nil {
		return nil, fmt.Errorf("failed to register get_mcp_version: %w", err)
	}
	if err := tools.RegisterScaffoldAddon(server); err != nil {
		return nil, fmt.Errorf("failed to register scaffold_addon: %w", err)
	}
	if err := tools.RegisterGenerateBedrockSnippet(server); err != nil {
		return nil, fmt.Errorf("failed to register generate_bedrock_snippet: %w", err)
	}
	if err := tools.RegisterGenerateUUID(server); err != nil {
		return nil, fmt.Errorf("failed to register generate_uuid: %w", err)
	}
	if err := tools.RegisterManifestDoctor(server); err != nil {
		return nil, fmt.Errorf("failed to register manifest_doctor: %w", err)
	}
	if err := tools.RegisterManifestFixup(server); err != nil {
		return nil, fmt.Errorf("failed to register manifest_fixup: %w", err)
	}
	if err := tools.RegisterDiffScriptAPIVersions(server, npmClient); err != nil {
		return nil, fmt.Errorf("failed to register diff_script_api_versions: %w", err)
	}

	// Register static resource: bedrock://docs/strict_rules
	if err := server.RegisterResource("bedrock://docs/strict_rules",
		"Bedrock Script API Strict Rules",
		"Bedrock Script API guardrails and syntax cheat sheet",
		"text/markdown",
		func() (*mcp.ResourceResponse, error) {
			return mcp.NewResourceResponse(
				mcp.NewTextEmbeddedResource("bedrock://docs/strict_rules", resources.StrictRules(), "text/markdown"),
			), nil
		}); err != nil {
		return nil, fmt.Errorf("failed to register strict_rules resource: %w", err)
	}

	return server, nil
}

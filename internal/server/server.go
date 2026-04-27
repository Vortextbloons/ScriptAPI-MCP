package server

import (
	"fmt"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/resources"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/tools"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/version"
)

// New creates and configures the MCP server with all tools and resources
func New() (*mcp.Server, error) {
	transport := stdio.NewStdioServerTransport()
	server := mcp.NewServer(transport,
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

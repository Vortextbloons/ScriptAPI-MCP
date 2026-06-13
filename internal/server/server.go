package server

import (
	"fmt"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/resources"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/tools"
	mcpstdio "github.com/isaac-org/Script-API-Helper-MCP/internal/transport"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/version"
)

// New creates and configures the MCP server with all tools and resources using stdio transport.
func New() (*mcp.Server, error) {
	return NewWithTransport(mcpstdio.NewStdioServerTransport())
}

// NewWithTransport creates and configures the MCP server with a custom transport.
func NewWithTransport(tr transport.Transport) (*mcp.Server, error) {
	server := mcp.NewServer(tr,
		mcp.WithName(version.Name),
		mcp.WithVersion(version.Current),
	)

	npmClient := npm.NewClient()

	if err := tools.RegisterResolveAPIEnvironment(server, npmClient); err != nil {
		return nil, fmt.Errorf("failed to register resolve_api_environment: %w", err)
	}
	if err := tools.RegisterSearchAPI(server, npmClient); err != nil {
		return nil, fmt.Errorf("failed to register search_api: %w", err)
	}
	if err := tools.RegisterManifest(server); err != nil {
		return nil, fmt.Errorf("failed to register manifest: %w", err)
	}
	if err := tools.RegisterVersionInfo(server); err != nil {
		return nil, fmt.Errorf("failed to register get_mcp_version: %w", err)
	}
	if err := tools.RegisterScaffoldAddon(server, npmClient); err != nil {
		return nil, fmt.Errorf("failed to register scaffold_addon: %w", err)
	}
	if err := tools.RegisterGenerateCode(server); err != nil {
		return nil, fmt.Errorf("failed to register generate_code: %w", err)
	}
	if err := tools.RegisterGenerateUUID(server); err != nil {
		return nil, fmt.Errorf("failed to register generate_uuid: %w", err)
	}
	if err := tools.RegisterDiffScriptAPIVersions(server, npmClient); err != nil {
		return nil, fmt.Errorf("failed to register diff_script_api_versions: %w", err)
	}
	if err := tools.RegisterDiagnoseWorkspace(server); err != nil {
		return nil, fmt.Errorf("failed to register diagnose_workspace: %w", err)
	}
	if err := tools.RegisterDistributeAddon(server); err != nil {
		return nil, fmt.Errorf("failed to register distribute_addon: %w", err)
	}
	if err := tools.RegisterBedrockReference(server); err != nil {
		return nil, fmt.Errorf("failed to register bedrock_reference: %w", err)
	}

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

	if err := server.RegisterResource("bedrock://docs/module_guide",
		"Bedrock Script API Module Guide",
		"Module selection and version guidance for Bedrock Script API",
		"text/markdown",
		func() (*mcp.ResourceResponse, error) {
			return mcp.NewResourceResponse(
				mcp.NewTextEmbeddedResource("bedrock://docs/module_guide", resources.ModuleGuide(), "text/markdown"),
			), nil
		}); err != nil {
		return nil, fmt.Errorf("failed to register module_guide resource: %w", err)
	}

	return server, nil
}

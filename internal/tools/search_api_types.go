package tools

import (
	"fmt"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

// SearchAPITypesInput is the input schema for Tool 3
type SearchAPITypesInput struct {
	Module  string `json:"module" mcp:"required,description='Module name (e.g. @minecraft/server)'"`
	Query   string `json:"query" mcp:"required,description='Class, interface, or namespace to look up (e.g. Player, ActionFormData, world.afterEvents)'"`
	Version string `json:"version" mcp:"required,description='Resolved npm version from Tool 1'"`
}

// RegisterSearchAPITypes registers Tool 3
func RegisterSearchAPITypes(server *mcp.Server, npmClient *npm.Client) error {
	return server.RegisterTool("search_api_types",
		"Queries specific TypeScript definitions from the live npm package. Returns only the requested symbol and its referenced types instead of the full .d.ts file.",
		func(args SearchAPITypesInput) (*mcp.ToolResponse, error) {
			return handleSearchAPITypes(args, npmClient)
		})
}

func handleSearchAPITypes(args SearchAPITypesInput, npmClient *npm.Client) (*mcp.ToolResponse, error) {
	// Fetch the .d.ts for this module+version
	dts, err := npmClient.FetchTypes(args.Module, args.Version)
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error fetching types: %v", err))), nil
	}

	// Extract the specific query
	result, err := npm.ExtractTypes(dts, args.Query)
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error extracting types: %v", err))), nil
	}

	return mcp.NewToolResponse(mcp.NewTextContent(result)), nil
}

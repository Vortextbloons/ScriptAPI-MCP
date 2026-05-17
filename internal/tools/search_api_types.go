package tools

import (
	"fmt"
	"strings"

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
	version := strings.TrimSpace(args.Version)
	if version == "" {
		return toolErrorResponse("INVALID_INPUT", "version is required", false, "Pass exact_npm_version from resolve_api_environment"), nil
	}

	if ok, err := npmClient.LookupExactVersion(args.Module, version); err == nil && !ok {
		candidates, cerr := npmClient.ListConcreteVersions(args.Module, version)
		if cerr != nil {
			return toolErrorResponse("VERSION_LOOKUP_FAILED", fmt.Sprintf("unable to verify version %q: %v", version, cerr), true, "Retry version lookup", "Use exact_npm_version from resolve_api_environment"), nil
		}
		if len(candidates) == 1 {
			version = candidates[0]
		} else {
			return toolErrorResponse("AMBIGUOUS_VERSION", fmt.Sprintf("version %q is not an exact publish", version), false, append([]string{"Use an exact publish version"}, candidates...)...), nil
		}
	} else if err != nil {
		return toolErrorResponse("VERSION_LOOKUP_FAILED", fmt.Sprintf("unable to verify version: %v", err), true, "Retry lookup", "Check npm registry availability"), nil
	}

	// Fetch the .d.ts for this module+version
	dts, err := npmClient.FetchTypes(args.Module, version)
	if err != nil {
		return toolErrorResponse("FETCH_TYPES_FAILED", fmt.Sprintf("error fetching types for %s@%s: %v", args.Module, version, err), true, "Retry with the same version", "Confirm module name is valid"), nil
	}

	// Extract the specific query
	result, err := npm.ExtractTypes(dts, args.Query)
	if err != nil {
		return toolErrorResponse("TYPE_EXTRACT_FAILED", fmt.Sprintf("error extracting types: %v", err), false, "Try a top-level symbol name like Player", "Try a shorter query without member chaining"), nil
	}

	return mcp.NewToolResponse(mcp.NewTextContent(result)), nil
}

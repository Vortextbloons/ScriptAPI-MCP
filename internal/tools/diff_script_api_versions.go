package tools

import (
	"encoding/json"
	"fmt"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/apidiff"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

type DiffScriptAPIVersionsInput struct {
	Module             string `json:"module" mcp:"required,description='Module name, e.g. @minecraft/server'"`
	FromVersion        string `json:"from_version" mcp:"required,description='Exact publish version for baseline (e.g. 2.7.0)'"`
	ToVersion          string `json:"to_version" mcp:"required,description='Exact publish version for target (e.g. 2.8.0-beta.1.26.20-preview.28)'"`
	IncludeNonBreaking bool   `json:"include_non_breaking" mcp:"description='Include additive/non-breaking changes in output'"`
	SymbolFilter       string `json:"symbol_filter" mcp:"description='Only show results containing this substring (case-insensitive)'"`
	MaxResults         int    `json:"max_results" mcp:"description='Cap results (default 50, max 200)'"`
}

func RegisterDiffScriptAPIVersions(server *mcp.Server, npmClient *npm.Client) error {
	return server.RegisterTool("diff_script_api_versions",
		"Diffs the TypeScript API surface between two exact versions of a @minecraft/* module. Both versions must be exact publish strings. If you supply a shorthand like 2.8.0-beta, an error will list the concrete versions available. Returns breaking changes, non-breaking changes, and possible renames.",
		func(args DiffScriptAPIVersionsInput) (*mcp.ToolResponse, error) {
			return handleDiffScriptAPIVersions(args, npmClient)
		})
}

func handleDiffScriptAPIVersions(args DiffScriptAPIVersionsInput, npmClient *npm.Client) (*mcp.ToolResponse, error) {
	if args.Module == "" {
		return toolErrorResponse("INVALID_INPUT", "module name is required", false), nil
	}

	fromExists, err := npmClient.LookupExactVersion(args.Module, args.FromVersion)
	if err != nil {
		return toolErrorResponse("VERSION_LOOKUP_FAILED", fmt.Sprintf("error checking from_version: %v", err), true), nil
	}
	if !fromExists {
		candidates, cerr := npmClient.ListConcreteVersions(args.Module, args.FromVersion)
		if cerr != nil {
			return toolErrorResponse("AMBIGUOUS_VERSION", fmt.Sprintf("version %q is not an exact publish and could not list candidates: %v", args.FromVersion, cerr), false), nil
		}
		if len(candidates) == 0 {
			return toolErrorResponse("AMBIGUOUS_VERSION", fmt.Sprintf("version %q is not an exact publish and no matching candidates found", args.FromVersion), false), nil
		}
		jsonCandidates, _ := json.MarshalIndent(candidates, "", "  ")
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Version %q is ambiguous. Use an exact publish. Matching concrete versions:\n%s", args.FromVersion, string(jsonCandidates)))), nil
	}

	toExists, err := npmClient.LookupExactVersion(args.Module, args.ToVersion)
	if err != nil {
		return toolErrorResponse("VERSION_LOOKUP_FAILED", fmt.Sprintf("error checking to_version: %v", err), true), nil
	}
	if !toExists {
		candidates, cerr := npmClient.ListConcreteVersions(args.Module, args.ToVersion)
		if cerr != nil {
			return toolErrorResponse("AMBIGUOUS_VERSION", fmt.Sprintf("version %q is not an exact publish and could not list candidates: %v", args.ToVersion, cerr), false), nil
		}
		if len(candidates) == 0 {
			return toolErrorResponse("AMBIGUOUS_VERSION", fmt.Sprintf("version %q is not an exact publish and no matching candidates found", args.ToVersion), false), nil
		}
		jsonCandidates, _ := json.MarshalIndent(candidates, "", "  ")
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Version %q is ambiguous. Use an exact publish. Matching concrete versions:\n%s", args.ToVersion, string(jsonCandidates)))), nil
	}

	fromDTS, err := npmClient.FetchTypes(args.Module, args.FromVersion)
	if err != nil {
		return toolErrorResponse("FETCH_TYPES_FAILED", fmt.Sprintf("error fetching types for %s@%s: %v", args.Module, args.FromVersion, err), true), nil
	}
	toDTS, err := npmClient.FetchTypes(args.Module, args.ToVersion)
	if err != nil {
		return toolErrorResponse("FETCH_TYPES_FAILED", fmt.Sprintf("error fetching types for %s@%s: %v", args.Module, args.ToVersion, err), true), nil
	}

	fromTable, err := apidiff.BuildSymbolTable(fromDTS, args.Module, args.FromVersion)
	if err != nil {
		return toolErrorResponse("SYMBOL_TABLE_FAILED", fmt.Sprintf("error building symbol table for %s@%s: %v", args.Module, args.FromVersion, err), false), nil
	}
	toTable, err := apidiff.BuildSymbolTable(toDTS, args.Module, args.ToVersion)
	if err != nil {
		return toolErrorResponse("SYMBOL_TABLE_FAILED", fmt.Sprintf("error building symbol table for %s@%s: %v", args.Module, args.ToVersion, err), false), nil
	}

	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}

	result := apidiff.CompareTables(fromTable, toTable, args.IncludeNonBreaking, args.SymbolFilter, maxResults)

	result.Module = args.Module
	result.RequestedFrom = args.FromVersion
	result.ResolvedFrom = args.FromVersion
	result.RequestedTo = args.ToVersion
	result.ResolvedTo = args.ToVersion
	result.FromVerified = true
	result.ToVerified = true

	apidiff.SortDiffResult(result)

	jsonOut, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
}

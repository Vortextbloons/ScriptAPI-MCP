package tools

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

type SearchAPIInput struct {
	Module           string   `json:"module" mcp:"description='source=registry: required @minecraft module name (e.g. @minecraft/server). source=local: optional when modules is supplied or when auto-scanning.'"`
	Version          string   `json:"version" mcp:"description='source=registry only: exact npm publish version (alternative to minecraft_version+channel)'"`
	MinecraftVersion string   `json:"minecraft_version" mcp:"description='source=registry only: target Minecraft game version (e.g. 1.21.70). Used with channel to resolve npm version automatically when version is not provided.'"`
	Channel          string   `json:"channel" mcp:"description='source=registry only: version channel when using minecraft_version: stable, beta, or preview (default beta)'"`
	Source           string   `json:"source" mcp:"description='Data source: registry (default, fetch from npm) or local (read from project_path/node_modules/@minecraft/*)'"`
	ProjectPath      string   `json:"project_path" mcp:"description='source=local only: project root containing node_modules. Required when source=local.'"`
	Modules          []string `json:"modules" mcp:"description='source=local only: explicit module list. If omitted, scans all installed @minecraft/*.'"`
	Query            string   `json:"query" mcp:"required,description='types mode: symbol name (e.g. Player, world.afterEvents). members/index modes: arbitrary substring to match in .d.ts lines.'"`
	Mode             string   `json:"mode" mcp:"description='Search mode: types (structured symbol extraction, default) | members (grep-style substring match) | index (lightweight export catalog, no bodies)'"`
	Limit            int      `json:"limit" mcp:"description='Max results (default 50, max 200)'"`
	Offset           int      `json:"offset" mcp:"description='source=local only: pagination offset (default 0)'"`
	ContextLines     int      `json:"context_lines" mcp:"description='source=registry members mode only: number of surrounding lines to include for each match (default 0)'"`
}

type SearchAPIMatch struct {
	LineNumber int      `json:"line_number"`
	Line       string   `json:"line"`
	Context    []string `json:"context,omitempty"`
}

func RegisterSearchAPI(server *mcp.Server, npmClient *npm.Client) error {
	return server.RegisterTool("search_api",
		"Searches TypeScript definitions for @minecraft/* packages. Two sources: 'registry' (default) fetches the live .d.ts from npm and requires an exact version or a Minecraft version + channel. 'local' reads from <project_path>/node_modules/@minecraft/* (offline, matches the user's installed version). Three modes: 'types' (structured symbol extraction, default), 'members' (grep-style substring match), and 'index' (lightweight export catalog, local only). Local calls accept an optional 'modules' list and 'offset' for pagination.",
		func(args SearchAPIInput) (*mcp.ToolResponse, error) {
			return handleSearchAPI(args, npmClient)
		})
}

func handleSearchAPI(args SearchAPIInput, npmClient *npm.Client) (*mcp.ToolResponse, error) {
	source := strings.ToLower(strings.TrimSpace(args.Source))
	if source == "" {
		source = "registry"
	}
	if source != "registry" && source != "local" {
		return toolErrorResponse("INVALID_INPUT", fmt.Sprintf("unknown source %q (use registry or local)", args.Source), false, "Omit 'source' to use the default (registry)"), nil
	}
	if source == "local" {
		return handleSearchAPILocal(args)
	}
	return handleSearchAPIRegistry(args, npmClient)
}

func handleSearchAPIRegistry(args SearchAPIInput, npmClient *npm.Client) (*mcp.ToolResponse, error) {
	version := strings.TrimSpace(args.Version)
	mcVersion := strings.TrimSpace(args.MinecraftVersion)
	channel := strings.TrimSpace(args.Channel)
	mode := strings.ToLower(strings.TrimSpace(args.Mode))
	if mode == "" {
		mode = "types"
	}
	if mode == "index" {
		return toolErrorResponse("INVALID_INPUT", "mode=index is only supported for source=local", false, "Use mode=members for grep-style matching", "Use mode=types for structured symbol extraction"), nil
	}

	module := strings.TrimSpace(args.Module)
	if module == "" {
		return toolErrorResponse("INVALID_INPUT", "module is required for source=registry", false, "Provide a module like @minecraft/server"), nil
	}
	args.Module = module

	if version == "" && mcVersion == "" {
		return toolErrorResponse("INVALID_INPUT", "either version or minecraft_version is required", false, "Provide exact_npm_version from resolve_api_environment", "Or provide minecraft_version (e.g. 1.21.70) with optional channel"), nil
	}

	if version == "" && mcVersion != "" {
		vm, err := npmClient.FetchVersionMatrix(args.Module)
		if err != nil {
			return toolErrorResponse("VERSION_RESOLVE_FAILED", fmt.Sprintf("unable to fetch version matrix for %s: %v", args.Module, err), true, "Retry with explicit version instead"), nil
		}
		resolved, err := npm.ResolveVersionForChannel(vm, mcVersion, channel)
		if err != nil {
			return toolErrorResponse("VERSION_RESOLVE_FAILED", fmt.Sprintf("no matching version for %s @ Minecraft %s (channel: %s): %v", args.Module, mcVersion, channel, err), false, "Try a different Minecraft version", "Try stable instead of beta", "Use resolve_api_environment with mode=list-versions to see available versions"), nil
		}
		version = resolved
	}

	if ok, err := npmClient.LookupExactVersion(args.Module, version); err == nil && !ok {
		candidates, cerr := npmClient.ListConcreteVersions(args.Module, version)
		if cerr != nil {
			return toolErrorResponse("VERSION_LOOKUP_FAILED", fmt.Sprintf("unable to verify version %q: %v", version, cerr), true, "Retry version lookup", "Use exact_npm_version from resolve_api_environment"), nil
		}
		if len(candidates) == 1 {
			version = candidates[0]
		} else {
			return toolErrorResponse("AMBIGUOUS_VERSION", fmt.Sprintf("version %q is not an exact publish", version), false, append([]string{"Use an exact publish version or specify minecraft_version"}, candidates...)...), nil
		}
	} else if err != nil {
		return toolErrorResponse("VERSION_LOOKUP_FAILED", fmt.Sprintf("unable to verify version: %v", err), true, "Retry lookup", "Check npm registry availability"), nil
	}

	dts, err := npmClient.FetchTypes(args.Module, version)
	if err != nil {
		return toolErrorResponse("FETCH_TYPES_FAILED", fmt.Sprintf("error fetching types for %s@%s: %v", args.Module, version, err), true, "Retry with the same version", "Confirm module name is valid"), nil
	}

	limit, lerr := clampLimit(args.Limit)
	if lerr != nil {
		return toolErrorResponse("INVALID_INPUT", lerr.Error(), false), nil
	}

	switch mode {
	case "members":
		q := strings.ToLower(strings.TrimSpace(args.Query))
		if q == "" {
			return toolErrorResponse("INVALID_INPUT", "query is required", false), nil
		}
		ctx := args.ContextLines
		if ctx < 0 {
			ctx = 0
		}
		lines := strings.Split(string(dts), "\n")
		matches := make([]SearchAPIMatch, 0)
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if strings.Contains(strings.ToLower(trimmed), q) {
				match := SearchAPIMatch{
					LineNumber: i + 1,
					Line:       trimmed,
				}
				if ctx > 0 {
					start := int(math.Max(0, float64(i-ctx)))
					end := int(math.Min(float64(len(lines)-1), float64(i+ctx)))
					context := make([]string, 0, end-start)
					for j := start; j <= end; j++ {
						if j == i {
							continue
						}
						context = append(context, lines[j])
					}
					match.Context = context
				}
				matches = append(matches, match)
				if len(matches) >= limit {
					break
				}
			}
		}
		if len(matches) == 0 {
			return toolErrorResponse("NO_MATCH", fmt.Sprintf("no members matched query %q", args.Query), false), nil
		}
		b, _ := json.MarshalIndent(map[string]any{
			"module":  args.Module,
			"version": version,
			"query":   args.Query,
			"mode":    "members",
			"count":   len(matches),
			"matches": matches,
		}, "", "  ")
		return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil

	default:
		result, err := npm.ExtractTypes(dts, args.Query)
		if err != nil {
			return toolErrorResponse("TYPE_EXTRACT_FAILED", fmt.Sprintf("error extracting types: %v", err), false, "Try a top-level symbol name like Player", "Try a shorter query without member chaining"), nil
		}
		b, _ := json.MarshalIndent(map[string]any{
			"module":  args.Module,
			"version": version,
			"query":   args.Query,
			"mode":    "types",
			"result":  result,
		}, "", "  ")
		return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
	}
}

// clampLimit normalizes a user-supplied limit value to the allowed range.
// Returns an error when the user requested a value above the hard cap.
func clampLimit(n int) (int, error) {
	const (
		defaultLimit = 50
		maxLimit     = 200
	)
	if n <= 0 {
		return defaultLimit, nil
	}
	if n > maxLimit {
		return 0, fmt.Errorf("limit %d exceeds max of %d", n, maxLimit)
	}
	return n, nil
}

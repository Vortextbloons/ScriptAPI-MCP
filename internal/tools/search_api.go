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
	Module          string `json:"module" mcp:"required,description='@minecraft module name (e.g. @minecraft/server)'"`
	Version         string `json:"version" mcp:"description='Exact npm publish version (alternative to minecraft_version+channel)'"`
	MinecraftVersion string `json:"minecraft_version" mcp:"description='Target Minecraft game version (e.g. 1.21.70). Used with channel to resolve npm version automatically when version is not provided.'"`
	Channel         string `json:"channel" mcp:"description='Version channel when using minecraft_version: stable, beta, or preview (default beta)'"`
	Query           string `json:"query" mcp:"required,description='types mode: symbol name (e.g. Player, world.afterEvents). members mode: arbitrary substring to match in .d.ts lines.'"`
	Mode            string `json:"mode" mcp:"description='Search mode: types (structured symbol extraction, default) or members (grep-style substring match)'"`
	Limit           int    `json:"limit" mcp:"description='Max results (default 50)'"`
	ContextLines    int    `json:"context_lines" mcp:"description='members mode only: number of surrounding lines to include for each match (default 0)'"`
}

type SearchAPIMatch struct {
	LineNumber int      `json:"line_number"`
	Line       string   `json:"line"`
	Context    []string `json:"context,omitempty"`
}

func RegisterSearchAPI(server *mcp.Server, npmClient *npm.Client) error {
	return server.RegisterTool("search_api",
		"Searches TypeScript definitions from a live @minecraft/* npm package. Use mode=types for structured symbol lookup (class/interface/namespace) or mode=members for grep-style substring matching across .d.ts lines. Accepts either an exact npm version or a Minecraft version + channel for automatic resolution.",
		func(args SearchAPIInput) (*mcp.ToolResponse, error) {
			return handleSearchAPI(args, npmClient)
		})
}

func handleSearchAPI(args SearchAPIInput, npmClient *npm.Client) (*mcp.ToolResponse, error) {
	version := strings.TrimSpace(args.Version)
	mcVersion := strings.TrimSpace(args.MinecraftVersion)
	channel := strings.TrimSpace(args.Channel)

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
			return toolErrorResponse("VERSION_RESOLVE_FAILED", fmt.Sprintf("no matching version for %s @ Minecraft %s (channel: %s): %v", args.Module, mcVersion, channel, err), false, "Try a different Minecraft version", "Try stable instead of beta", "Use list_api_versions to see available versions"), nil
		}
		version = resolved
	}

	mode := strings.ToLower(strings.TrimSpace(args.Mode))
	if mode == "" {
		mode = "types"
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

	limit := args.Limit
	if limit <= 0 {
		limit = 50
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

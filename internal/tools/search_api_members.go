package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

type SearchAPIMembersInput struct {
	Module  string `json:"module" mcp:"required,description='@minecraft module name'"`
	Version string `json:"version" mcp:"required,description='Exact npm publish version'"`
	Query   string `json:"query" mcp:"required,description='Substring to match in .d.ts lines'"`
	Limit   int    `json:"limit" mcp:"description='max results (default 50)'"`
}

func RegisterSearchAPIMembers(server *mcp.Server, npmClient *npm.Client) error {
	return server.RegisterTool("search_api_members",
		"Searches API members by text match against TypeScript definitions for a module version.",
		func(args SearchAPIMembersInput) (*mcp.ToolResponse, error) {
			dts, err := npmClient.FetchTypes(args.Module, args.Version)
			if err != nil {
				return toolErrorResponse("FETCH_TYPES_FAILED", err.Error(), true), nil
			}
			q := strings.ToLower(strings.TrimSpace(args.Query))
			if q == "" {
				return toolErrorResponse("INVALID_INPUT", "query is required", false), nil
			}
			limit := args.Limit
			if limit <= 0 {
				limit = 50
			}
			lines := strings.Split(string(dts), "\n")
			matches := make([]string, 0)
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					continue
				}
				if strings.Contains(strings.ToLower(trimmed), q) {
					matches = append(matches, trimmed)
					if len(matches) >= limit {
						break
					}
				}
			}
			b, _ := json.MarshalIndent(map[string]any{"module": args.Module, "version": args.Version, "query": args.Query, "count": len(matches), "matches": matches}, "", "  ")
			if len(matches) == 0 {
				return toolErrorResponse("NO_MATCH", fmt.Sprintf("no members matched query %q", args.Query), false), nil
			}
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

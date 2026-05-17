package tools

import (
	"encoding/json"
	"sort"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

type ListAPIVersionsInput struct {
	Module  string `json:"module" mcp:"required,description='@minecraft module name'"`
	Channel string `json:"channel" mcp:"description='stable|beta|preview|all'"`
	Limit   int    `json:"limit" mcp:"description='max number of versions to return (default 30)'"`
}

func RegisterListAPIVersions(server *mcp.Server, npmClient *npm.Client) error {
	return server.RegisterTool("list_api_versions",
		"Lists available npm publish versions for a @minecraft module with channel filtering.",
		func(args ListAPIVersionsInput) (*mcp.ToolResponse, error) {
			vm, err := npmClient.FetchVersionMatrix(args.Module)
			if err != nil {
				return toolErrorResponse("VERSION_LIST_FAILED", err.Error(), true), nil
			}
			limit := args.Limit
			if limit <= 0 {
				limit = 30
			}
			ch := args.Channel
			if ch == "" {
				ch = "all"
			}
			out := make([]string, 0, len(vm.Versions))
			for _, v := range vm.Versions {
				if ch == "all" || matchesVersionChannel(v, ch) {
					out = append(out, v)
				}
			}
			sort.Slice(out, func(i, j int) bool { return out[i] > out[j] })
			if len(out) > limit {
				out = out[:limit]
			}
			b, _ := json.MarshalIndent(map[string]any{"module": args.Module, "channel": ch, "versions": out}, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

func matchesVersionChannel(version, channel string) bool {
	switch channel {
	case "stable":
		return !strings.Contains(version, "-beta") && !strings.Contains(version, "-preview") && !strings.Contains(version, "-rc")
	case "beta":
		return strings.Contains(version, "-beta") && !strings.Contains(version, "-preview")
	case "preview":
		return strings.Contains(version, "-preview")
	default:
		return true
	}
}

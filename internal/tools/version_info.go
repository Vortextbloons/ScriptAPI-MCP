package tools

import (
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/version"
)

type VersionInfoOutput struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type VersionInfoInput struct{}

func RegisterVersionInfo(server *mcp.Server) error {
	return server.RegisterTool("get_mcp_version",
		"Returns the current MCP name (includes version) and version string so the AI can report it back accurately.",
		func(args VersionInfoInput) (*mcp.ToolResponse, error) {
			out := VersionInfoOutput{Name: version.DisplayName(), Version: version.Current}
			b, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

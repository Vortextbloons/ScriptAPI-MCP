package tools

import (
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/snippets"
)

type GenerateBedrockSnippetInput struct {
	SnippetType     string `json:"snippet_type" mcp:"required,description='Snippet type: beforeEvents.playerBreakBlock, afterEvents.playerSpawn, worldInitialize, custom_item_template, custom_block_template, script_event_handler'"`
	Language        string `json:"language" mcp:"description='javascript or typescript (default javascript)'"`
	Name            string `json:"name" mcp:"description='Optional identifier for generated code (e.g. custom component name)'"`
	ModuleVersion   string `json:"module_version" mcp:"description='Optional version string injected as comment'"`
	IncludeComments bool   `json:"include_comments" mcp:"description='Whether to include descriptive JSDoc/TSDoc comments (default false)'"`
}

func RegisterGenerateBedrockSnippet(server *mcp.Server) error {
	return server.RegisterTool("generate_bedrock_snippet",
		"Generates a Bedrock Script API code snippet for common event subscriptions and component templates. Supports JavaScript and TypeScript output.",
		func(args GenerateBedrockSnippetInput) (*mcp.ToolResponse, error) {
			out, err := snippets.GenerateSnippet(args.SnippetType, args.Language, args.Name, args.ModuleVersion, args.IncludeComments)
			if err != nil {
				return mcp.NewToolResponse(mcp.NewTextContent("Error: " + err.Error())), nil
			}
			b, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

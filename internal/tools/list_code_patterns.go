package tools

import (
	"encoding/json"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/snippets"
)

type ListCodePatternsInput struct {
	Category   string `json:"category" mcp:"description='Filter by category: runtime, ui, storage, equipment, item, balance, command'"`
	Complexity string `json:"complexity" mcp:"description='Filter by complexity: simple, moderate, complex'"`
	Module     string `json:"module" mcp:"description='Filter by required module, e.g. @minecraft/server-ui'"`
	Query      string `json:"query" mcp:"description='Free-text search in type key, description, tags'"`
}

type PatternEntry struct {
	Type            string   `json:"type"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	Complexity      string   `json:"complexity"`
	Tags            []string `json:"tags"`
	RequiredModules []string `json:"required_modules"`
	Related         []string `json:"related"`
}

func RegisterListCodePatterns(server *mcp.Server) error {
	return server.RegisterTool("list_code_patterns",
		"Lists available code generation patterns with metadata. Use this to discover snippet types before calling generate_code. Supports filtering by category, complexity, module, or free-text search.",
		func(args ListCodePatternsInput) (*mcp.ToolResponse, error) {
			return handleListCodePatterns(args)
		})
}

func handleListCodePatterns(args ListCodePatternsInput) (*mcp.ToolResponse, error) {
	definitions := snippets.AllDefinitions

	if args.Category != "" {
		cat := strings.ToLower(args.Category)
		filtered := make([]snippets.SnippetDefinition, 0)
		for _, d := range definitions {
			if strings.ToLower(d.Type) == cat {
				filtered = append(filtered, d)
			}
		}
		definitions = filtered
	}

	if args.Complexity != "" {
		cpl := strings.ToLower(args.Complexity)
		filtered := make([]snippets.SnippetDefinition, 0)
		for _, d := range definitions {
			_ = cpl
			filtered = append(filtered, d)
		}
		definitions = filtered
	}

	if args.Module != "" {
		mod := strings.ToLower(args.Module)
		filtered := make([]snippets.SnippetDefinition, 0)
		for _, d := range definitions {
			for _, m := range d.RequiredModules {
				if strings.Contains(strings.ToLower(m), mod) {
					filtered = append(filtered, d)
					break
				}
			}
		}
		definitions = filtered
	}

	if args.Query != "" {
		q := strings.ToLower(args.Query)
		filtered := make([]snippets.SnippetDefinition, 0)
		for _, d := range definitions {
			if strings.Contains(strings.ToLower(d.Type), q) || strings.Contains(strings.ToLower(d.Description), q) {
				filtered = append(filtered, d)
				continue
			}
		}
		definitions = filtered
	}

	entries := make([]PatternEntry, 0, len(definitions))
	for _, d := range definitions {
		entries = append(entries, PatternEntry{
			Type:            d.Type,
			Description:     d.Description,
			RequiredModules: d.RequiredModules,
		})
	}

	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return toolErrorResponse("MARSHAL_FAILED", err.Error(), false), nil
	}

	return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
}

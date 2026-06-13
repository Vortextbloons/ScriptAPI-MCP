package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/snippets"
)

type GenerateCodeInput struct {
	Mode            string `json:"mode" mcp:"description='generate (default) to produce code, list to discover available snippet types'"`
	SnippetType     string `json:"snippet_type" mcp:"description='generate mode: snippet type key (e.g. beforeEvents.playerBreakBlock, action_form, runtime.plugin_registry). Use mode=list first to discover types.'"`
	Title           string `json:"title" mcp:"description='generate mode: form title (action_form, modal_form, message_form only)'"`
	Identifier      string `json:"identifier" mcp:"description='generate mode: custom component identifier (custom_item only), e.g. myaddon:wand'"`
	Language        string `json:"language" mcp:"description='generate mode: javascript or typescript (default javascript)'"`
	Name            string `json:"name" mcp:"description='generate mode: optional name for template placeholders'"`
	ModuleVersion   string `json:"module_version" mcp:"description='generate mode: optional version string injected as comment'"`
	IncludeComments bool   `json:"include_comments" mcp:"description='generate mode: include descriptive comments (default false)'"`
	Category        string `json:"category" mcp:"description='list mode: filter by category (runtime, ui, storage, equipment, item, balance, command)'"`
	Complexity      string `json:"complexity" mcp:"description='list mode: filter by complexity (simple, moderate, complex)'"`
	Module          string `json:"module" mcp:"description='list mode: filter by required module, e.g. @minecraft/server-ui'"`
	Query           string `json:"query" mcp:"description='list mode: free-text search in type key, description, or tags'"`
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

func RegisterGenerateCode(server *mcp.Server) error {
	return server.RegisterTool("generate_code",
		"Generates Bedrock Script API boilerplate or lists available patterns. Use mode=list to discover snippet types (filter by category, complexity, module, or query). Use mode=generate (default) with snippet_type to produce JavaScript or TypeScript code for events, forms, custom items/blocks, and advanced patterns.",
		func(args GenerateCodeInput) (*mcp.ToolResponse, error) {
			return handleGenerateCode(args)
		})
}

func handleGenerateCode(args GenerateCodeInput) (*mcp.ToolResponse, error) {
	mode := strings.ToLower(strings.TrimSpace(args.Mode))
	if mode == "" {
		mode = "generate"
	}

	switch mode {
	case "list":
		return handleListCodePatterns(args)
	case "generate":
		return handleGenerateCodeSnippet(args)
	default:
		return toolErrorResponse("INVALID_INPUT", fmt.Sprintf("unknown mode %q (use list or generate)", args.Mode), false), nil
	}
}

func handleListCodePatterns(args GenerateCodeInput) (*mcp.ToolResponse, error) {
	definitions := snippets.AllDefinitions

	if args.Category != "" {
		cat := strings.ToLower(args.Category)
		filtered := make([]snippets.SnippetDefinition, 0)
		for _, d := range definitions {
			if strings.ToLower(d.Category) == cat {
				filtered = append(filtered, d)
			}
		}
		definitions = filtered
	}

	if args.Complexity != "" {
		cpl := strings.ToLower(args.Complexity)
		filtered := make([]snippets.SnippetDefinition, 0)
		for _, d := range definitions {
			if strings.ToLower(d.Complexity) == cpl {
				filtered = append(filtered, d)
			}
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
			for _, tag := range d.Tags {
				if strings.Contains(strings.ToLower(tag), q) {
					filtered = append(filtered, d)
					break
				}
			}
		}
		definitions = filtered
	}

	entries := make([]PatternEntry, 0, len(definitions))
	for _, d := range definitions {
		entries = append(entries, PatternEntry{
			Type:            d.Type,
			Description:     d.Description,
			Category:        d.Category,
			Complexity:      d.Complexity,
			Tags:            d.Tags,
			RequiredModules: d.RequiredModules,
			Related:         d.Related,
		})
	}

	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return toolErrorResponse("MARSHAL_FAILED", err.Error(), false), nil
	}

	return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
}

func handleGenerateCodeSnippet(args GenerateCodeInput) (*mcp.ToolResponse, error) {
	if strings.TrimSpace(args.SnippetType) == "" {
		return toolErrorResponse("INVALID_INPUT", "snippet_type is required in generate mode (use mode=list to discover types)", false), nil
	}

	lang := args.Language
	if lang == "" {
		lang = "javascript"
	}

	switch args.SnippetType {
	case "action_form", "modal_form", "message_form":
		if args.Title == "" {
			return toolErrorResponse("INVALID_INPUT", "title is required for form snippets", false), nil
		}
		code, err := generateFormCode(args.SnippetType, args.Title, lang)
		if err != nil {
			return toolErrorResponse("GENERATE_FAILED", err.Error(), false), nil
		}
		b, _ := json.MarshalIndent(map[string]string{"language": lang, "code": code, "snippet_type": args.SnippetType}, "", "  ")
		return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil

	case "custom_item":
		id := args.Identifier
		if id == "" {
			return toolErrorResponse("INVALID_INPUT", "identifier is required for custom_item (e.g. myaddon:wand)", false), nil
		}
		code := fmt.Sprintf("import { world } from \"@minecraft/server\";\n\nworld.afterEvents.worldInitialize.subscribe((event) => {\n  event.propertyRegistry.registerCustomComponent(\"%s\", {\n    onUse({ source }) {\n      source.sendMessage(\"Custom item used\");\n    },\n  });\n});\n", id)
		if lang == "typescript" {
			code = fmt.Sprintf("import { world, type WorldInitializeAfterEvent, type ItemComponentUseEvent } from \"@minecraft/server\";\n\nworld.afterEvents.worldInitialize.subscribe((event: WorldInitializeAfterEvent): void => {\n  event.propertyRegistry.registerCustomComponent(\"%s\", {\n    onUse({ source }: ItemComponentUseEvent): void {\n      source.sendMessage(\"Custom item used\");\n    },\n  });\n});\n", id)
		}
		b, _ := json.MarshalIndent(map[string]string{"language": lang, "code": code, "snippet_type": "custom_item"}, "", "  ")
		return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil

	default:
		out, err := snippets.GenerateSnippet(args.SnippetType, lang, args.Name, args.ModuleVersion, args.IncludeComments)
		if err != nil {
			return mcp.NewToolResponse(mcp.NewTextContent("Error: " + err.Error())), nil
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
	}
}

func generateFormCode(formType, title, lang string) (string, error) {
	var code string
	switch formType {
	case "action_form":
		code = fmt.Sprintf("import { ActionFormData } from \"@minecraft/server-ui\";\n\nconst form = new ActionFormData().title(\"%s\").body(\"Choose an option\").button(\"Option 1\");\n", title)
	case "modal_form":
		code = fmt.Sprintf("import { ModalFormData } from \"@minecraft/server-ui\";\n\nconst form = new ModalFormData().title(\"%s\").textField(\"Value\", \"enter...\", \"\");\n", title)
	case "message_form":
		code = fmt.Sprintf("import { MessageFormData } from \"@minecraft/server-ui\";\n\nconst form = new MessageFormData().title(\"%s\").body(\"Confirm?\").button1(\"Yes\").button2(\"No\");\n", title)
	default:
		return "", fmt.Errorf("unknown form type: %s", formType)
	}
	if strings.EqualFold(lang, "typescript") {
		code += "\n// show with: await form.show(player);\n"
	}
	return code, nil
}

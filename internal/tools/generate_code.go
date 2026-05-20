package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/snippets"
)

type GenerateCodeInput struct {
	SnippetType     string `json:"snippet_type" mcp:"required,description='Code type. Event snippets: beforeEvents.playerBreakBlock, afterEvents.playerSpawn, worldInitialize, custom_item_template, custom_block_template, script_event_handler. UI forms: action_form, modal_form, message_form. Custom item: custom_item. Advanced patterns: runtime.plugin_registry, runtime.background_scheduler, runtime.profile_cache, runtime.cooldown_manager, ui.action_form_wizard, interaction.item_interaction_handler, storage.dynamic_property_store, storage.world_config, equipment.equipment_scanner, item.lore_builder, balance.scaled_value, command.custom_slash_command'"`
	Title           string `json:"title" mcp:"description='Form title (for action_form, modal_form, message_form only)'"`
	Identifier      string `json:"identifier" mcp:"description='Custom component identifier (for custom_item only), e.g. myaddon:wand'"`
	Language        string `json:"language" mcp:"description='javascript or typescript (default javascript)'"`
	Name            string `json:"name" mcp:"description='Optional name for template placeholders (e.g. custom component name)'"`
	ModuleVersion   string `json:"module_version" mcp:"description='Optional version string injected as comment'"`
	IncludeComments bool   `json:"include_comments" mcp:"description='Whether to include descriptive comments (default false)'"`
}

func RegisterGenerateCode(server *mcp.Server) error {
	return server.RegisterTool("generate_code",
		"Generates Bedrock Script API boilerplate code. Supports event subscriptions, custom item/block components, worldInitialize, UI forms (action/modal/message), and script event handlers in JavaScript or TypeScript.",
		func(args GenerateCodeInput) (*mcp.ToolResponse, error) {
			return handleGenerateCode(args)
		})
}

func handleGenerateCode(args GenerateCodeInput) (*mcp.ToolResponse, error) {
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

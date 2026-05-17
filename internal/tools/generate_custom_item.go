package tools

import (
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"
)

type GenerateCustomItemInput struct {
	Identifier string `json:"identifier" mcp:"required,description='Custom component identifier, e.g. myaddon:wand'"`
	Language   string `json:"language" mcp:"description='javascript or typescript (default javascript)'"`
}

func RegisterGenerateCustomItem(server *mcp.Server) error {
	return server.RegisterTool("generate_custom_item",
		"Generates custom item component boilerplate for worldInitialize registration.",
		func(args GenerateCustomItemInput) (*mcp.ToolResponse, error) {
			lang := args.Language
			if lang == "" {
				lang = "javascript"
			}
			code := "import { world } from \"@minecraft/server\";\n\nworld.afterEvents.worldInitialize.subscribe((event) => {\n  event.propertyRegistry.registerCustomComponent(\"" + args.Identifier + "\", {\n    onUse({ source }) {\n      source.sendMessage(\"Custom item used\");\n    },\n  });\n});\n"
			if lang == "typescript" {
				code = "import { world, type WorldInitializeAfterEvent, type ItemComponentUseEvent } from \"@minecraft/server\";\n\nworld.afterEvents.worldInitialize.subscribe((event: WorldInitializeAfterEvent): void => {\n  event.propertyRegistry.registerCustomComponent(\"" + args.Identifier + "\", {\n    onUse({ source }: ItemComponentUseEvent): void {\n      source.sendMessage(\"Custom item used\");\n    },\n  });\n});\n"
			}
			b, _ := json.MarshalIndent(map[string]string{"language": lang, "code": code}, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

package tools

import (
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"
)

type GenerateUIFormInput struct {
	FormType string `json:"form_type" mcp:"required,description='action|modal|message'"`
	Title    string `json:"title" mcp:"required,description='Form title'"`
	Language string `json:"language" mcp:"description='javascript or typescript (default javascript)'"`
}

func RegisterGenerateUIForm(server *mcp.Server) error {
	return server.RegisterTool("generate_ui_form",
		"Generates @minecraft/server-ui form boilerplate.",
		func(args GenerateUIFormInput) (*mcp.ToolResponse, error) {
			lang := args.Language
			if lang == "" {
				lang = "javascript"
			}
			code := ""
			switch args.FormType {
			case "action":
				code = "import { ActionFormData } from \"@minecraft/server-ui\";\n\nconst form = new ActionFormData().title(\"" + args.Title + "\").body(\"Choose an option\").button(\"Option 1\");\n"
			case "modal":
				code = "import { ModalFormData } from \"@minecraft/server-ui\";\n\nconst form = new ModalFormData().title(\"" + args.Title + "\").textField(\"Value\", \"enter...\", \"\");\n"
			case "message":
				code = "import { MessageFormData } from \"@minecraft/server-ui\";\n\nconst form = new MessageFormData().title(\"" + args.Title + "\").body(\"Confirm?\").button1(\"Yes\").button2(\"No\");\n"
			default:
				return toolErrorResponse("INVALID_INPUT", "form_type must be action|modal|message", false), nil
			}
			if lang == "typescript" {
				code += "\n// show with: await form.show(player);\n"
			}
			b, _ := json.MarshalIndent(map[string]string{"language": lang, "code": code}, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

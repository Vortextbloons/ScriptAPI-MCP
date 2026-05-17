package tools

import (
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"
)

type TroubleshootPackNotLoadingInput struct {
	ProjectPath string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
}

func RegisterTroubleshootPackNotLoading(server *mcp.Server) error {
	return server.RegisterTool("troubleshoot_pack_not_loading",
		"Diagnoses common reasons Bedrock packs fail to load and returns actionable checks.",
		func(args TroubleshootPackNotLoadingInput) (*mcp.ToolResponse, error) {
			val, err := validateAddonWorkspace(args.ProjectPath)
			if err != nil {
				return toolErrorResponse("TROUBLESHOOT_FAILED", err.Error(), false), nil
			}
			checks := []string{
				"Ensure manifests are valid JSON",
				"Ensure UUIDs are unique",
				"Ensure script entry file exists",
				"Ensure module versions are compatible",
				"Ensure pack folders are deployed to com.mojang development directories",
			}
			b, _ := json.MarshalIndent(map[string]any{"valid": val.Valid, "findings": val.Findings, "recommended_checks": checks}, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

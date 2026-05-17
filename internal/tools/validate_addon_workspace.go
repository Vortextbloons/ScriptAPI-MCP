package tools

import (
	"encoding/json"
	"path/filepath"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"
)

type ValidateAddonWorkspaceInput struct {
	ProjectPath string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
}

type WorkspaceFinding struct {
	Severity string `json:"severity"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
	Fixable  bool   `json:"fixable"`
}

type ValidateAddonWorkspaceOutput struct {
	Valid    bool               `json:"valid"`
	Findings []WorkspaceFinding `json:"findings"`
}

func RegisterValidateAddonWorkspace(server *mcp.Server) error {
	return server.RegisterTool("validate_addon_workspace",
		"Validates a Bedrock addon workspace structure, entrypoints, and dependency alignment.",
		func(args ValidateAddonWorkspaceInput) (*mcp.ToolResponse, error) {
			out, err := validateAddonWorkspace(args.ProjectPath)
			if err != nil {
				return toolErrorResponse("WORKSPACE_VALIDATE_FAILED", err.Error(), false), nil
			}
			b, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

func validateAddonWorkspace(projectPath string) (*ValidateAddonWorkspaceOutput, error) {
	inspect, err := inspectAddonWorkspace(projectPath)
	if err != nil {
		return nil, err
	}

	findings := make([]WorkspaceFinding, 0)
	if inspect.SourceEntrypoint == "" {
		findings = append(findings, WorkspaceFinding{Severity: "warning", Rule: "missing_source_entry", Message: "missing src/main.js or src/main.ts", Fixable: true})
	}
	if inspect.Entrypoint == "" {
		findings = append(findings, WorkspaceFinding{Severity: "error", Rule: "missing_script_entry", Message: "manifest has no script module entry", Fixable: true})
	}
	for _, w := range inspect.Warnings {
		findings = append(findings, WorkspaceFinding{Severity: "error", Rule: "invalid_entrypoint", Message: w, Fixable: true})
	}

	if inspect.Language == "typescript" && !fileExists(filepath.Join(strings.TrimSpace(projectPath), "tsconfig.json")) {
		findings = append(findings, WorkspaceFinding{Severity: "warning", Rule: "missing_tsconfig", Message: "TypeScript workspace is missing tsconfig.json", Fixable: true})
	}

	valid := true
	for _, f := range findings {
		if f.Severity == "error" {
			valid = false
			break
		}
	}

	return &ValidateAddonWorkspaceOutput{Valid: valid, Findings: findings}, nil
}

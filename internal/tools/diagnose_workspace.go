package tools

import (
	"encoding/json"
	"path/filepath"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"
)

type DiagnoseWorkspaceInput struct {
	ProjectPath string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
	Mode        string `json:"mode" mcp:"description='Output mode: validate (default), score, or troubleshoot'"`
}

func RegisterDiagnoseWorkspace(server *mcp.Server) error {
	return server.RegisterTool("diagnose_workspace",
		"Validates a Bedrock addon workspace structure, computes health score, or troubleshoots pack loading issues. Use mode: validate (default), score, or troubleshoot.",
		func(args DiagnoseWorkspaceInput) (*mcp.ToolResponse, error) {
			mode := strings.ToLower(strings.TrimSpace(args.Mode))
			if mode == "" {
				mode = "validate"
			}
			if mode != "validate" && mode != "score" && mode != "troubleshoot" {
				mode = "validate"
			}

			val, err := validateAddonWorkspace(args.ProjectPath)
			if err != nil {
				return toolErrorResponse("DIAGNOSE_FAILED", err.Error(), false), nil
			}

			switch mode {
			case "score":
				score := 100
				for _, f := range val.Findings {
					if f.Severity == "error" {
						score -= 25
					} else {
						score -= 10
					}
				}
				if score < 0 {
					score = 0
				}
				status := "excellent"
				switch {
				case score < 50:
					status = "poor"
				case score < 75:
					status = "fair"
				case score < 90:
					status = "good"
				}
				b, _ := json.MarshalIndent(map[string]any{"score": score, "status": status, "findings": val.Findings}, "", "  ")
				return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil

			case "troubleshoot":
				checks := []string{
					"Ensure manifests are valid JSON",
					"Ensure UUIDs are unique",
					"Ensure script entry file exists",
					"Ensure module versions are compatible",
					"Ensure pack folders are deployed to com.mojang development directories",
				}
				b, _ := json.MarshalIndent(map[string]any{"valid": val.Valid, "findings": val.Findings, "recommended_checks": checks}, "", "  ")
				return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil

			default:
				b, _ := json.MarshalIndent(val, "", "  ")
				return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
			}
		})
}

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

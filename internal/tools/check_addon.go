package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"
)

type CheckAddonInput struct {
	ProjectPath string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
	Mode        string `json:"mode" mcp:"description='check (default), build, score, or troubleshoot. build explicitly runs npm run build.'"`
}
type BuildDiagnostic struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}
type CheckAddonOutput struct {
	Valid          bool               `json:"valid"`
	BuildRan       bool               `json:"build_ran"`
	BuildCommand   string             `json:"build_command,omitempty"`
	OutputExpected string             `json:"output_expected,omitempty"`
	OutputExists   bool               `json:"output_exists"`
	Findings       []WorkspaceFinding `json:"findings"`
	Diagnostics    []BuildDiagnostic  `json:"diagnostics,omitempty"`
}

func RegisterCheckAddon(server *mcp.Server) error {
	return server.RegisterTool("check_addon", "Checks a Bedrock addon workspace. mode=check (default) validates structure, configuration, and manifest output; mode=build explicitly runs npm run build; mode=score returns a health score; mode=troubleshoot adds load-failure guidance.", func(a CheckAddonInput) (*mcp.ToolResponse, error) { return handleCheckAddon(a) })
}

func handleCheckAddon(args CheckAddonInput) (*mcp.ToolResponse, error) {
	mode := strings.ToLower(strings.TrimSpace(args.Mode))
	if mode == "" {
		mode = "check"
	}
	if mode != "check" && mode != "build" && mode != "score" && mode != "troubleshoot" {
		return toolErrorResponse("INVALID_INPUT", "mode must be check, build, score, or troubleshoot", false), nil
	}
	inspect, err := inspectAddonWorkspace(args.ProjectPath)
	if err != nil {
		return toolErrorResponse("CHECK_FAILED", err.Error(), false), nil
	}
	validation, err := validateAddonWorkspace(args.ProjectPath)
	if err != nil {
		return toolErrorResponse("CHECK_FAILED", err.Error(), false), nil
	}
	out := CheckAddonOutput{Valid: validation.Valid, Findings: validation.Findings}
	if inspect.Entrypoint != "" {
		out.OutputExpected = filepath.ToSlash(filepath.Join(inspect.BPSource, inspect.Entrypoint))
		out.OutputExists = fileExists(filepath.Join(args.ProjectPath, filepath.FromSlash(inspect.BPSource), filepath.FromSlash(inspect.Entrypoint)))
	}
	if inspect.Language == "typescript" && !fileExists(filepath.Join(args.ProjectPath, "tsconfig.json")) {
		out.Findings = append(out.Findings, WorkspaceFinding{Severity: "error", Rule: "missing_tsconfig", Message: "TypeScript source requires tsconfig.json", Fixable: true})
		out.Valid = false
	}
	packagePath := filepath.Join(args.ProjectPath, "package.json")
	if fileExists(packagePath) {
		out.BuildCommand = "npm run build"
	} else if inspect.Language == "typescript" {
		out.Findings = append(out.Findings, WorkspaceFinding{Severity: "warning", Rule: "missing_build_script", Message: "TypeScript workspace has no package.json build script", Fixable: true})
	}
	if mode == "build" {
		if out.BuildCommand == "" {
			return toolErrorResponse("BUILD_NOT_CONFIGURED", "no package.json build script found", false), nil
		}
		cmd := exec.Command("npm", "run", "build")
		cmd.Dir = args.ProjectPath
		data, runErr := cmd.CombinedOutput()
		out.BuildRan = true
		if strings.TrimSpace(string(data)) != "" {
			out.Diagnostics = append(out.Diagnostics, BuildDiagnostic{Severity: "info", Message: strings.TrimSpace(string(data))})
		}
		if runErr != nil {
			out.Valid = false
			out.Diagnostics = append(out.Diagnostics, BuildDiagnostic{Severity: "error", Message: fmt.Sprintf("build failed: %v", runErr)})
		}
		out.OutputExists = out.OutputExpected != "" && fileExists(filepath.Join(args.ProjectPath, filepath.FromSlash(out.OutputExpected)))
	}
	if out.OutputExpected != "" && !out.OutputExists {
		out.Valid = false
		out.Findings = append(out.Findings, WorkspaceFinding{Severity: "error", Rule: "missing_compiled_entry", Message: "manifest script entrypoint is missing: " + out.OutputExpected, Fixable: true})
	}
	if mode == "score" {
		score := 100
		for _, f := range out.Findings {
			if f.Severity == "error" {
				score -= 25
			} else {
				score -= 10
			}
		}
		if score < 0 {
			score = 0
		}
		b, _ := json.MarshalIndent(map[string]any{"score": score, "valid": out.Valid, "findings": out.Findings}, "", "  ")
		return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
	}
	if mode == "troubleshoot" {
		out.Diagnostics = append(out.Diagnostics, BuildDiagnostic{Severity: "info", Message: "Check unique UUIDs, manifest JSON, module compatibility, script output, and deployment under com.mojang development packs."})
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
}

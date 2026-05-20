package tools

import (
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifestdoctor"
)

type ManifestCheckInput struct {
	ManifestJSON             string   `json:"manifest_json" mcp:"required,description='Raw manifest JSON string to validate or fix'"`
	Mode                     string   `json:"mode" mcp:"description='Operation mode: diagnose (default) to find issues, fix to apply auto-fixes from doctor findings'"`
	FixAllFixable            bool     `json:"fix_all_fixable" mcp:"description='fix mode only: apply all fixable rules at once (default true). Ignored in diagnose mode.'"`
	Fixes                    []string `json:"fixes" mcp:"description='fix mode only: specific fix IDs from doctor output to apply. Ignored if fix_all_fixable is true.'"`
	ProjectPath              string   `json:"project_path" mcp:"description='diagnose mode only: optional path to project root for node_modules checks'"`
	PackKindHint             string   `json:"pack_kind_hint" mcp:"description='diagnose mode only: behavior, resource, or unknown (auto-detected if empty)'"`
	CheckLocalModules        bool     `json:"check_local_modules" mcp:"description='diagnose mode only: if true, validates against installed node_modules'"`
	MinEngineVersionOverride []int    `json:"min_engine_version_override" mcp:"description='Expected min_engine_version for either mode (default [1,21,60])'"`
}

func RegisterManifestCheck(server *mcp.Server) error {
	return server.RegisterTool("manifest_check",
		"Validates a Bedrock manifest.json for common issues (missing fields, invalid UUIDs, deprecated modules) or auto-fixes them. Use mode=diagnose first, then mode=fix with the returned fix IDs.",
		func(args ManifestCheckInput) (*mcp.ToolResponse, error) {
			mode := args.Mode
			if mode == "" {
				mode = "diagnose"
			}

			opts := &manifestdoctor.DoctorOptions{}
			if len(args.MinEngineVersionOverride) == 3 {
				opts.MinEngineVersion = args.MinEngineVersionOverride
			}

			switch mode {
			case "fix":
				opts.MinEngineVersion = []int{1, 21, 60}
				if len(args.MinEngineVersionOverride) == 3 {
					opts.MinEngineVersion = args.MinEngineVersionOverride
				}
				result := manifestdoctor.RunFixup(args.ManifestJSON, args.Fixes, args.FixAllFixable, opts)
				jsonOut, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil

			default:
				opts.CheckLocalModules = args.CheckLocalModules
				opts.ProjectPath = args.ProjectPath
				result := manifestdoctor.RunDoctor(args.ManifestJSON, opts)
				jsonOut, _ := json.MarshalIndent(result, "", "  ")
				return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
			}
		})
}



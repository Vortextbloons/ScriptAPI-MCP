package tools

import (
	"encoding/json"
	"fmt"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifestdoctor"
)

type ManifestInput struct {
	ManifestJSON             string   `json:"manifest_json" mcp:"description='Raw manifest JSON string (required for all modes)'"`
	Mode                     string   `json:"mode" mcp:"description='diagnose (default) to find issues, fix to apply auto-fixes, sync-deps to add/remove script module dependencies'"`
	FixAllFixable            bool     `json:"fix_all_fixable" mcp:"description='fix mode: apply all fixable rules at once (default true)'"`
	Fixes                    []string `json:"fixes" mcp:"description='fix mode: specific fix IDs from doctor output (ignored if fix_all_fixable is true)'"`
	ProjectPath              string   `json:"project_path" mcp:"description='diagnose mode: optional project root for node_modules checks'"`
	PackKindHint             string   `json:"pack_kind_hint" mcp:"description='diagnose mode: behavior, resource, or unknown'"`
	CheckLocalModules        bool     `json:"check_local_modules" mcp:"description='diagnose mode: validate against installed node_modules'"`
	MinEngineVersionOverride []int    `json:"min_engine_version_override" mcp:"description='Expected min_engine_version (default [1,21,60])'"`
	AddedModules             []string `json:"added_modules" mcp:"description='sync-deps mode: modules to add (e.g. @minecraft/server-net)'"`
	RemovedModules           []string `json:"removed_modules" mcp:"description='sync-deps mode: modules to remove'"`
}

func RegisterManifest(server *mcp.Server) error {
	return server.RegisterTool("manifest",
		"Operates on Bedrock manifest.json. Use mode=diagnose (default) to find issues, mode=fix to apply auto-fixes from doctor findings, mode=sync-deps to safely add or remove script module dependencies.",
		func(args ManifestInput) (*mcp.ToolResponse, error) {
			mode := args.Mode
			if mode == "" {
				mode = "diagnose"
			}

			if args.ManifestJSON == "" {
				return toolErrorResponse("INVALID_INPUT", "manifest_json is required", false), nil
			}

			switch mode {
			case "fix":
				return handleManifestFix(args)
			case "sync-deps", "sync_deps":
				return handleSyncManifestDependencies(ManifestInput{
					ManifestJSON:   args.ManifestJSON,
					AddedModules:   args.AddedModules,
					RemovedModules: args.RemovedModules,
				})
			default:
				return handleManifestDiagnose(args)
			}
		})
}

func handleManifestDiagnose(args ManifestInput) (*mcp.ToolResponse, error) {
	opts := &manifestdoctor.DoctorOptions{}
	if len(args.MinEngineVersionOverride) == 3 {
		opts.MinEngineVersion = args.MinEngineVersionOverride
	}
	opts.CheckLocalModules = args.CheckLocalModules
	opts.ProjectPath = args.ProjectPath

	result := manifestdoctor.RunDoctor(args.ManifestJSON, opts)
	jsonOut, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
}

func handleManifestFix(args ManifestInput) (*mcp.ToolResponse, error) {
	opts := &manifestdoctor.DoctorOptions{MinEngineVersion: []int{1, 21, 60}}
	if len(args.MinEngineVersionOverride) == 3 {
		opts.MinEngineVersion = args.MinEngineVersionOverride
	}

	result := manifestdoctor.RunFixup(args.ManifestJSON, args.Fixes, args.FixAllFixable, opts)
	jsonOut, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
}

func handleSyncManifestDependencies(args ManifestInput) (*mcp.ToolResponse, error) {
	m, err := manifest.ParseManifest(args.ManifestJSON)
	if err != nil {
		return toolErrorResponse("MANIFEST_PARSE_FAILED", fmt.Sprintf("error parsing manifest: %v", err), false), nil
	}

	if err := manifest.UpdateDependencies(&m, args.AddedModules, args.RemovedModules); err != nil {
		return toolErrorResponse("DEPENDENCY_VALIDATION_FAILED", fmt.Sprintf("validation error: %v", err), false), nil
	}

	updated, err := manifest.FormatManifest(m)
	if err != nil {
		return toolErrorResponse("MANIFEST_FORMAT_FAILED", fmt.Sprintf("error formatting manifest: %v", err), false), nil
	}

	return mcp.NewToolResponse(mcp.NewTextContent(updated)), nil
}

package tools

import (
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifestdoctor"
)

type ManifestFixupInput struct {
	ManifestJSON             string   `json:"manifest_json" mcp:"required,description='Raw manifest JSON string to fix'"`
	Fixes                    []string `json:"fixes" mcp:"description='Specific fix IDs from doctor output to apply. Ignored if fix_all_fixable is true'"`
	FixAllFixable            bool     `json:"fix_all_fixable" mcp:"description='Apply all fixable rules at once (default true)'"`
	MinEngineVersionOverride []int    `json:"min_engine_version_override" mcp:"description='Value to use for min_engine_version fixes (default [1,21,60])'"`
}

func RegisterManifestFixup(server *mcp.Server) error {
	return server.RegisterTool("manifest_fixup",
		"Applies auto-fixes to a Bedrock manifest.json based on doctor findings. Use after manifest_doctor to automatically correct fixable issues like missing UUIDs, wrong version format, or deprecated modules.",
		func(args ManifestFixupInput) (*mcp.ToolResponse, error) {
			opts := &manifestdoctor.DoctorOptions{
				MinEngineVersion: []int{1, 21, 60},
			}
			if len(args.MinEngineVersionOverride) == 3 {
				opts.MinEngineVersion = args.MinEngineVersionOverride
			}
			result := manifestdoctor.RunFixup(args.ManifestJSON, args.Fixes, args.FixAllFixable, opts)
			jsonOut, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
		})
}

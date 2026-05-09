package tools

import (
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifestdoctor"
)

type ManifestDoctorInput struct {
	ManifestJSON             string `json:"manifest_json" mcp:"required,description='Raw manifest JSON string to validate'"`
	ProjectPath              string `json:"project_path" mcp:"description='Optional path to project root for node_modules checks'"`
	PackKindHint             string `json:"pack_kind_hint" mcp:"description='behavior, resource, or unknown (auto-detected if empty)'"`
	CheckLocalModules        bool   `json:"check_local_modules" mcp:"description='If true, validates against installed node_modules'"`
	MinEngineVersionOverride []int  `json:"min_engine_version_override" mcp:"description='Expected min_engine_version (default [1,21,60])'"`
}

func RegisterManifestDoctor(server *mcp.Server) error {
	return server.RegisterTool("manifest_doctor",
		"Validates a Bedrock manifest.json for common issues: missing fields, invalid UUIDs, deprecated modules, duplicate dependencies, and structural problems. Returns structured findings with fix IDs for use with manifest_fixup.",
		func(args ManifestDoctorInput) (*mcp.ToolResponse, error) {
			opts := &manifestdoctor.DoctorOptions{
				CheckLocalModules: args.CheckLocalModules,
				ProjectPath:       args.ProjectPath,
			}
			if len(args.MinEngineVersionOverride) == 3 {
				opts.MinEngineVersion = args.MinEngineVersionOverride
			}
			result := manifestdoctor.RunDoctor(args.ManifestJSON, opts)
			jsonOut, _ := json.MarshalIndent(result, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
		})
}

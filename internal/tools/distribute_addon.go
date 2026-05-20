package tools

import (
	"encoding/json"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"
)

type DistributeAddonInput struct {
	ProjectPath string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
	Action      string `json:"action" mcp:"description='Operation: package (create .mcaddon, default), deploy (copy to com.mojang), or both'"`
	OutputPath  string `json:"output_path" mcp:"description='package/both mode: path for .mcaddon output file'"`
	MCDevPath   string `json:"mcdev_path" mcp:"description='deploy/both mode: path to com.mojang folder'"`
	BPDestName  string `json:"bp_dest_name" mcp:"description='deploy/both mode: destination behavior pack folder name'"`
	RPDestName  string `json:"rp_dest_name" mcp:"description='deploy/both mode: destination resource pack folder name'"`
	DryRun      bool   `json:"dry_run" mcp:"description='If true, previews the operation without writing files'"`
}

func RegisterDistributeAddon(server *mcp.Server) error {
	return server.RegisterTool("distribute_addon",
		"Packages an addon workspace into a .mcaddon archive, deploys it to Minecraft development folders, or both. Use action=package (default), deploy, or both.",
		func(args DistributeAddonInput) (*mcp.ToolResponse, error) {
			return handleDistributeAddon(args)
		})
}

func handleDistributeAddon(args DistributeAddonInput) (*mcp.ToolResponse, error) {
	projectPath := strings.TrimSpace(args.ProjectPath)
	if projectPath == "" {
		return toolErrorResponse("INVALID_INPUT", "project_path is required", false), nil
	}

	action := strings.ToLower(strings.TrimSpace(args.Action))
	if action == "" {
		action = "package"
	}

	var results map[string]any

	switch action {
	case "deploy":
		mcdev := strings.TrimSpace(args.MCDevPath)
		if mcdev == "" {
			return toolErrorResponse("INVALID_INPUT", "mcdev_path is required for deploy mode", false), nil
		}
		deployArgs := DeployAddonInput{
			ProjectPath: projectPath,
			MCDevPath:   mcdev,
			BPDestName:  args.BPDestName,
			RPDestName:  args.RPDestName,
			DryRun:      args.DryRun,
		}
		out, err := deployAddon(deployArgs)
		if err != nil {
			return toolErrorResponse("DEPLOY_FAILED", err.Error(), false), nil
		}
		results = map[string]any{"action": "deploy", "result": out}

	case "both":
		outputPath := strings.TrimSpace(args.OutputPath)
		if outputPath == "" {
			return toolErrorResponse("INVALID_INPUT", "output_path is required for package mode", false), nil
		}
		mcdev := strings.TrimSpace(args.MCDevPath)
		if mcdev == "" {
			return toolErrorResponse("INVALID_INPUT", "mcdev_path is required for deploy mode", false), nil
		}

		pkgArgs := PackageAddonInput{
			ProjectPath: projectPath,
			OutputPath:  outputPath,
			DryRun:      args.DryRun,
		}
		pkgOut, err := packageAddon(pkgArgs)
		if err != nil {
			return toolErrorResponse("PACKAGE_FAILED", err.Error(), false), nil
		}

		deployArgs := DeployAddonInput{
			ProjectPath: projectPath,
			MCDevPath:   mcdev,
			BPDestName:  args.BPDestName,
			RPDestName:  args.RPDestName,
			DryRun:      args.DryRun,
		}
		depOut, err := deployAddon(deployArgs)
		if err != nil {
			return toolErrorResponse("DEPLOY_FAILED", err.Error(), false), nil
		}

		results = map[string]any{"action": "both", "package": pkgOut, "deploy": depOut}

	default:
		outputPath := strings.TrimSpace(args.OutputPath)
		if outputPath == "" {
			return toolErrorResponse("INVALID_INPUT", "output_path is required for package mode", false), nil
		}
		pkgArgs := PackageAddonInput{
			ProjectPath: projectPath,
			OutputPath:  outputPath,
			DryRun:      args.DryRun,
		}
		out, err := packageAddon(pkgArgs)
		if err != nil {
			return toolErrorResponse("PACKAGE_FAILED", err.Error(), false), nil
		}
		results = map[string]any{"action": "package", "result": out}
	}

	b, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
}



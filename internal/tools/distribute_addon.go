package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/models"
	mcp "github.com/metoro-io/mcp-golang"
)

const devSuffix = "-dev"

type DistributeAddonInput struct {
	ProjectPath string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
	Action      string `json:"action" mcp:"description='Operation: package (create .mcaddon, default), deploy (copy to com.mojang), or both'"`
	OutputPath  string `json:"output_path" mcp:"description='package/both mode: path for .mcaddon output file'"`
	MCDevPath   string `json:"mcdev_path" mcp:"description='deploy/both mode: path to com.mojang folder'"`
	BPDestName  string `json:"bp_dest_name" mcp:"description='deploy/both mode: destination behavior pack folder name'"`
	RPDestName  string `json:"rp_dest_name" mcp:"description='deploy/both mode: destination resource pack folder name'"`
	DryRun      bool   `json:"dry_run" mcp:"description='If true, previews the operation without writing files'"`
	DevPack     *bool  `json:"dev_pack" mcp:"description='Required for package/both. Ask the user before packaging: true adds -dev to manifest names and .mcaddon filename; false strips -dev for production release. Ignored for deploy-only.'"`
}

type devSuffixEntry struct {
	Original string `json:"original"`
	Final    string `json:"final"`
	Changed  bool   `json:"changed"`
}

type devSuffixReport struct {
	Requested string          `json:"requested"`
	BP        *devSuffixEntry `json:"bp,omitempty"`
	RP        *devSuffixEntry `json:"rp,omitempty"`
	Source    string          `json:"source"`
}

func RegisterDistributeAddon(server *mcp.Server) error {
	return server.RegisterTool("distribute_addon",
		"Packages an addon workspace into a .mcaddon archive, deploys it to Minecraft development folders, or both. Use action=package (default), deploy, or both. For package/both, ask the user 'Is this a dev pack?' (same as npm run bundle): dev_pack=true adds -dev to manifest header.name and the .mcaddon filename; dev_pack=false strips -dev for a production release. Source manifests are never modified.",
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

	packagesAddon := action == "package" || action == "both"
	var effectivePath string
	var report *devSuffixReport
	var restore func()

	if packagesAddon {
		if args.DevPack == nil {
			return toolErrorResponse("INVALID_INPUT", "Ask the user whether this is a dev pack, then pass dev_pack=true or dev_pack=false", false), nil
		}
		var err error
		effectivePath, report, restore, err = prepareDevSuffix(projectPath, *args.DevPack, args.DryRun)
		if err != nil {
			return toolErrorResponse("MANIFEST_PATCH_FAILED", err.Error(), false), nil
		}
	} else {
		effectivePath = projectPath
	}
	if restore != nil {
		defer restore()
	}

	var results map[string]any

	switch action {
	case "deploy":
		mcdev := strings.TrimSpace(args.MCDevPath)
		if mcdev == "" {
			return toolErrorResponse("INVALID_INPUT", "mcdev_path is required for deploy mode", false), nil
		}
		deployArgs := DeployAddonInput{
			ProjectPath: effectivePath,
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
		outputPath = applyDevSuffixToOutputPath(outputPath, *args.DevPack)
		mcdev := strings.TrimSpace(args.MCDevPath)
		if mcdev == "" {
			return toolErrorResponse("INVALID_INPUT", "mcdev_path is required for deploy mode", false), nil
		}

		pkgArgs := PackageAddonInput{
			ProjectPath: effectivePath,
			OutputPath:  outputPath,
			DryRun:      args.DryRun,
		}
		pkgOut, err := packageAddon(pkgArgs)
		if err != nil {
			return toolErrorResponse("PACKAGE_FAILED", err.Error(), false), nil
		}

		deployArgs := DeployAddonInput{
			ProjectPath: effectivePath,
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
		outputPath = applyDevSuffixToOutputPath(outputPath, *args.DevPack)
		pkgArgs := PackageAddonInput{
			ProjectPath: effectivePath,
			OutputPath:  outputPath,
			DryRun:      args.DryRun,
		}
		out, err := packageAddon(pkgArgs)
		if err != nil {
			return toolErrorResponse("PACKAGE_FAILED", err.Error(), false), nil
		}
		results = map[string]any{"action": "package", "result": out}
	}

	if report != nil {
		results["dev_suffix"] = report
	}

	b, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
}

func prepareDevSuffix(projectPath string, devPack bool, dryRun bool) (string, *devSuffixReport, func(), error) {
	manifestPaths := collectManifestPaths(projectPath)
	if len(manifestPaths) == 0 {
		return projectPath, nil, nil, nil
	}

	entries := make([]devSuffixEntry, 0, len(manifestPaths))
	parsed := make([]models.Manifest, 0, len(manifestPaths))

	hasName := false
	for _, mp := range manifestPaths {
		raw, err := os.ReadFile(mp)
		if err != nil {
			return "", nil, nil, fmt.Errorf("read %s: %w", mp, err)
		}
		m, err := manifest.ParseManifest(string(raw))
		if err != nil {
			return "", nil, nil, fmt.Errorf("parse %s: %w", mp, err)
		}
		parsed = append(parsed, m)
		if strings.TrimSpace(m.Header.Name) != "" {
			hasName = true
		}
	}

	if !hasName {
		return projectPath, nil, nil, nil
	}

	requested := "non-dev"
	if devPack {
		requested = "dev"
	}

	report := &devSuffixReport{Requested: requested, Source: projectPath}

	anyChange := false
	for i := range parsed {
		current := strings.TrimSpace(parsed[i].Header.Name)
		final := applyDevSuffix(current, devPack)
		entry := devSuffixEntry{Original: current, Final: final, Changed: final != current}
		entries = append(entries, entry)
		if i == 0 {
			report.BP = &entries[i]
		} else {
			report.RP = &entries[i]
		}
		if entry.Changed {
			parsed[i].Header.Name = final
			anyChange = true
		}
	}

	if !anyChange {
		return projectPath, report, nil, nil
	}

	if dryRun {
		return projectPath, report, nil, nil
	}

	stagingDir, err := os.MkdirTemp("", "script-api-helper-staging-")
	if err != nil {
		return "", nil, nil, fmt.Errorf("create staging dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(stagingDir) }

	if err := copyDir(projectPath, stagingDir); err != nil {
		cleanup()
		return "", nil, nil, fmt.Errorf("stage workspace: %w", err)
	}

	for i, mp := range manifestPaths {
		rel, err := filepath.Rel(projectPath, mp)
		if err != nil {
			cleanup()
			return "", nil, nil, fmt.Errorf("relpath %s: %w", mp, err)
		}
		stagedPath := filepath.Join(stagingDir, rel)
		formatted, err := manifest.FormatManifest(parsed[i])
		if err != nil {
			cleanup()
			return "", nil, nil, fmt.Errorf("format %s: %w", mp, err)
		}
		if err := os.WriteFile(stagedPath, []byte(formatted), 0o644); err != nil {
			cleanup()
			return "", nil, nil, fmt.Errorf("write staged %s: %w", mp, err)
		}
	}

	return stagingDir, report, cleanup, nil
}

func collectManifestPaths(projectPath string) []string {
	var paths []string
	for _, dir := range []string{"behavior_pack", "resource_pack"} {
		mp := filepath.Join(projectPath, dir, "manifest.json")
		if fileExists(mp) {
			paths = append(paths, mp)
		}
	}
	return paths
}

func applyDevSuffix(name string, isDev bool) string {
	trimmed := strings.TrimSpace(name)
	if isDev {
		if strings.HasSuffix(trimmed, devSuffix) {
			return trimmed
		}
		return trimmed + devSuffix
	}
	if strings.HasSuffix(trimmed, devSuffix) {
		return strings.TrimSuffix(trimmed, devSuffix)
	}
	return trimmed
}

func applyDevSuffixToOutputPath(outputPath string, isDev bool) string {
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return outputPath
	}
	dir := filepath.Dir(outputPath)
	base := filepath.Base(outputPath)
	if !strings.HasSuffix(strings.ToLower(base), ".mcaddon") {
		return applyDevSuffix(outputPath, isDev)
	}
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(dir, applyDevSuffix(stem, isDev)+".mcaddon")
}

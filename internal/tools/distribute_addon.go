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

// bpLayoutCandidate is a relative path under the project root that may contain
// a behavior or resource pack manifest.
type bpLayoutCandidate struct {
	// kind is "bp" or "rp"
	kind string
	// rel is the slash-separated path relative to the project root
	rel string
}

// defaultBP and defaultRP are the conventional roots the rest of the tool
// already assumes. They are checked first and have priority 0 in the
// auto-detect helper.
var defaultBP = []string{"behavior_pack"}
var defaultRP = []string{"resource_pack"}
var defaultScripts = []string{"dist", "build"}

// alternatePackRoots lists additional candidate folders checked after the
// defaults. Each entry is tried in order until a manifest.json is found.
var alternatePackRoots = []string{"static", "src", "packs", "addon"}

type PackLayout struct {
	BPSource      string `json:"bp_source"`
	RPSource      string `json:"rp_source"`
	ScriptsSource string `json:"scripts_source,omitempty"`
}

type DistributeAddonInput struct {
	ProjectPath    string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
	Action         string `json:"action" mcp:"description='Operation: package (create .mcaddon, default), deploy (copy to com.mojang), or both'"`
	OutputPath     string `json:"output_path" mcp:"description='package/both mode: path for .mcaddon output file'"`
	MCDevPath      string `json:"mcdev_path" mcp:"description='deploy/both mode: path to com.mojang folder'"`
	BPDestName     string `json:"bp_dest_name" mcp:"description='deploy/both mode: destination behavior pack folder name'"`
	RPDestName     string `json:"rp_dest_name" mcp:"description='deploy/both mode: destination resource pack folder name'"`
	DryRun         bool   `json:"dry_run" mcp:"description='If true, previews the operation without writing files'"`
	BPSource       string `json:"bp_source" mcp:"description='Override BP source folder relative to project_path (e.g. static/bp). Auto-detected if empty.'"`
	RPSource       string `json:"rp_source" mcp:"description='Override RP source folder relative to project_path (e.g. static/rp). Auto-detected if empty.'"`
	ScriptsSource  string `json:"scripts_source" mcp:"description='Folder whose contents are merged into the BP at scripts/ during packaging (e.g. dist). Auto-detected if empty; skipped if not found.'"`
	ManifestSuffix *bool  `json:"manifest_suffix" mcp:"description='Required for package/both. Ask the user first: true adds -dev to header.name and .mcaddon filename; false strips -dev for production. Source manifests are never modified.'"`
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
		"Packages an addon workspace into a .mcaddon archive, deploys it to Minecraft development folders, or both. Use action=package (default), deploy, or both. Auto-detects BP/RP/scripts source folders (behavior_pack, static/bp, src/bp, packs/bp, addon/bp) and merges a scripts_source (e.g. dist/) into the BP during packaging. For package/both, ask the user 'Is this a dev pack?' then pass manifest_suffix=true (adds -dev) or false (strips -dev). Source manifests are never modified.",
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

	// Resolve layout (BP/RP/scripts roots) once up front so the report and
	// both sub-operations agree.
	layout, err := resolvePackLayout(projectPath, args.BPSource, args.RPSource, args.ScriptsSource)
	if err != nil {
		return toolErrorResponse("INVALID_INPUT", err.Error(), false), nil
	}
	if layout.BPSource == "" {
		return toolErrorResponse("BP_NOT_FOUND",
			"could not locate a behavior pack manifest; checked behavior_pack/, static/bp/, src/bp/, packs/bp/, addon/bp/. Pass bp_source to override.",
			false,
			"Set bp_source to the BP folder relative to project_path",
			"Verify the project layout matches one of the supported conventions",
		), nil
	}

	packagesAddon := action == "package" || action == "both"

	var effectivePath string
	var report *devSuffixReport
	var restore func()
	var devPack bool

	if packagesAddon {
		if args.ManifestSuffix == nil {
			return toolErrorResponse("INVALID_INPUT", "Ask the user whether this is a dev pack, then pass manifest_suffix=true or manifest_suffix=false", false), nil
		}
		devPack = *args.ManifestSuffix
		effectivePath, report, restore, err = prepareDevSuffix(projectPath, layout, devPack, args.DryRun)
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
	results = map[string]any{"layout": layout}

	switch action {
	case "deploy":
		mcdev := strings.TrimSpace(args.MCDevPath)
		if mcdev == "" {
			return toolErrorResponse("INVALID_INPUT", "mcdev_path is required for deploy mode", false), nil
		}
		deployArgs := DeployAddonInput{
			ProjectPath:    effectivePath,
			MCDevPath:      mcdev,
			BPSource:       layout.BPSource,
			RPSource:       layout.RPSource,
			ScriptsSource:  layout.ScriptsSource,
			BPDestName:     args.BPDestName,
			RPDestName:     args.RPDestName,
			DryRun:         args.DryRun,
		}
		out, err := deployAddon(deployArgs)
		if err != nil {
			return toolErrorResponse("DEPLOY_FAILED", err.Error(), false), nil
		}
		results["action"] = "deploy"
		results["result"] = out

	case "both":
		outputPath := strings.TrimSpace(args.OutputPath)
		if outputPath == "" {
			return toolErrorResponse("INVALID_INPUT", "output_path is required for package mode", false), nil
		}
		outputPath = applyDevSuffixToOutputPath(outputPath, devPack)
		mcdev := strings.TrimSpace(args.MCDevPath)
		if mcdev == "" {
			return toolErrorResponse("INVALID_INPUT", "mcdev_path is required for deploy mode", false), nil
		}

		pkgArgs := PackageAddonInput{
			ProjectPath:   effectivePath,
			OutputPath:    outputPath,
			BPSource:      layout.BPSource,
			RPSource:      layout.RPSource,
			ScriptsSource: layout.ScriptsSource,
			DryRun:        args.DryRun,
		}
		pkgOut, err := packageAddon(pkgArgs)
		if err != nil {
			return toolErrorResponse("PACKAGE_FAILED", err.Error(), false), nil
		}

		deployArgs := DeployAddonInput{
			ProjectPath:   effectivePath,
			MCDevPath:     mcdev,
			BPSource:      layout.BPSource,
			RPSource:      layout.RPSource,
			ScriptsSource: layout.ScriptsSource,
			BPDestName:    args.BPDestName,
			RPDestName:    args.RPDestName,
			DryRun:        args.DryRun,
		}
		depOut, err := deployAddon(deployArgs)
		if err != nil {
			return toolErrorResponse("DEPLOY_FAILED", err.Error(), false), nil
		}

		results["action"] = "both"
		results["package"] = pkgOut
		results["deploy"] = depOut

	default:
		outputPath := strings.TrimSpace(args.OutputPath)
		if outputPath == "" {
			return toolErrorResponse("INVALID_INPUT", "output_path is required for package mode", false), nil
		}
		outputPath = applyDevSuffixToOutputPath(outputPath, devPack)
		pkgArgs := PackageAddonInput{
			ProjectPath:   effectivePath,
			OutputPath:    outputPath,
			BPSource:      layout.BPSource,
			RPSource:      layout.RPSource,
			ScriptsSource: layout.ScriptsSource,
			DryRun:        args.DryRun,
		}
		out, err := packageAddon(pkgArgs)
		if err != nil {
			return toolErrorResponse("PACKAGE_FAILED", err.Error(), false), nil
		}
		results["action"] = "package"
		results["result"] = out
	}

	if report != nil {
		results["dev_suffix"] = report
	}

	b, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
}

// resolvePackLayout determines the BP/RP/scripts source folders relative to
// projectPath. Overrides win over auto-detection.
func resolvePackLayout(projectPath, bpOverride, rpOverride, scriptsOverride string) (PackLayout, error) {
	layout := PackLayout{}

	if bpOverride != "" {
		if err := assertSafeRelative(bpOverride); err != nil {
			return layout, err
		}
		abs := filepath.Join(projectPath, filepath.FromSlash(bpOverride))
		if !fileExists(filepath.Join(abs, "manifest.json")) {
			return layout, fmt.Errorf("bp_source %q does not contain a manifest.json", bpOverride)
		}
		layout.BPSource = filepath.ToSlash(bpOverride)
	} else {
		bp, err := autoDetectPackRoot(projectPath, "bp", defaultBP)
		if err != nil {
			return layout, err
		}
		layout.BPSource = bp
	}

	if rpOverride != "" {
		if err := assertSafeRelative(rpOverride); err != nil {
			return layout, err
		}
		abs := filepath.Join(projectPath, filepath.FromSlash(rpOverride))
		if !fileExists(filepath.Join(abs, "manifest.json")) {
			return layout, fmt.Errorf("rp_source %q does not contain a manifest.json", rpOverride)
		}
		layout.RPSource = filepath.ToSlash(rpOverride)
	} else {
		rp, err := autoDetectPackRoot(projectPath, "rp", defaultRP)
		if err != nil {
			return layout, err
		}
		layout.RPSource = rp
	}

	if scriptsOverride != "" {
		if err := assertSafeRelative(scriptsOverride); err != nil {
			return layout, err
		}
		abs := filepath.Join(projectPath, filepath.FromSlash(scriptsOverride))
		if !fileExists(abs) {
			return layout, fmt.Errorf("scripts_source %q does not exist", scriptsOverride)
		}
		layout.ScriptsSource = filepath.ToSlash(scriptsOverride)
	} else {
		// Auto-detect: try default scripts sources in order, but only if the
		// directory exists. Do not error if none is found.
		for _, cand := range defaultScripts {
			abs := filepath.Join(projectPath, cand)
			if fileExists(abs) {
				layout.ScriptsSource = cand
				break
			}
		}
	}

	return layout, nil
}

// autoDetectPackRoot looks for a manifest.json at a sequence of candidate
// locations. The first default is checked first, then the standard
// alternate roots with the kind suffix appended.
func autoDetectPackRoot(projectPath, kind string, defaults []string) (string, error) {
	for _, d := range defaults {
		abs := filepath.Join(projectPath, d)
		if fileExists(filepath.Join(abs, "manifest.json")) {
			return d, nil
		}
	}
	for _, root := range alternatePackRoots {
		cand := root + "/" + kind
		abs := filepath.Join(projectPath, filepath.FromSlash(cand))
		if fileExists(filepath.Join(abs, "manifest.json")) {
			return cand, nil
		}
	}
	return "", nil
}

// assertSafeRelative blocks path traversal via "..".
func assertSafeRelative(rel string) error {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return nil
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, p := range parts {
		if p == ".." || p == "." && len(parts) > 1 {
			return fmt.Errorf("path %q contains invalid segment", rel)
		}
	}
	return nil
}

func prepareDevSuffix(projectPath string, layout PackLayout, devPack bool, dryRun bool) (string, *devSuffixReport, func(), error) {
	manifestPaths := make([]string, 0, 2)
	if layout.BPSource != "" {
		manifestPaths = append(manifestPaths, filepath.Join(projectPath, filepath.FromSlash(layout.BPSource), "manifest.json"))
	}
	if layout.RPSource != "" {
		manifestPaths = append(manifestPaths, filepath.Join(projectPath, filepath.FromSlash(layout.RPSource), "manifest.json"))
	}
	if len(manifestPaths) == 0 {
		return projectPath, nil, nil, nil
	}

	parsed := make([]models.Manifest, 0, len(manifestPaths))

	hasName := false
	for _, mp := range manifestPaths {
		if !fileExists(mp) {
			continue
		}
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
		// BP first, then RP (matches the order manifestPaths was built).
		if layout.BPSource != "" && i == 0 {
			report.BP = &entry
		} else if layout.RPSource != "" {
			report.RP = &entry
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

	pi := 0
	for _, mp := range manifestPaths {
		if !fileExists(mp) {
			continue
		}
		rel, err := filepath.Rel(projectPath, mp)
		if err != nil {
			cleanup()
			return "", nil, nil, fmt.Errorf("relpath %s: %w", mp, err)
		}
		stagedPath := filepath.Join(stagingDir, rel)
		formatted, err := manifest.FormatManifest(parsed[pi])
		if err != nil {
			cleanup()
			return "", nil, nil, fmt.Errorf("format %s: %w", mp, err)
		}
		if err := os.WriteFile(stagedPath, []byte(formatted), 0o644); err != nil {
			cleanup()
			return "", nil, nil, fmt.Errorf("write staged %s: %w", mp, err)
		}
		pi++
	}

	return stagingDir, report, cleanup, nil
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

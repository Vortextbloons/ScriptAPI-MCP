package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
)

type InspectAddonWorkspaceOutput struct {
	ProjectPath      string   `json:"project_path"`
	HasBehaviorPack  bool     `json:"has_behavior_pack"`
	HasResourcePack  bool     `json:"has_resource_pack"`
	Language         string   `json:"language"`
	Entrypoint       string   `json:"entrypoint,omitempty"`
	SourceEntrypoint string   `json:"source_entrypoint,omitempty"`
	Modules          []string `json:"modules"`
	Warnings         []string `json:"warnings,omitempty"`
}

func inspectAddonWorkspace(projectPath string) (*InspectAddonWorkspaceOutput, error) {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return nil, fmt.Errorf("project_path is required")
	}

	bpManifestPath := filepath.Join(projectPath, "behavior_pack", "manifest.json")
	rpManifestPath := filepath.Join(projectPath, "resource_pack", "manifest.json")

	hasBP := fileExists(bpManifestPath)
	hasRP := fileExists(rpManifestPath)
	if !hasBP {
		return nil, fmt.Errorf("behavior pack manifest not found at %s", bpManifestPath)
	}

	bpRaw, err := os.ReadFile(bpManifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read behavior pack manifest: %w", err)
	}
	bp, err := manifest.ParseManifest(string(bpRaw))
	if err != nil {
		return nil, fmt.Errorf("failed to parse behavior pack manifest: %w", err)
	}

	modules := make([]string, 0, len(bp.Dependencies))
	for _, dep := range bp.Dependencies {
		if dep.ModuleName != "" {
			modules = append(modules, dep.ModuleName)
		}
	}

	entry := ""
	for _, mod := range bp.Modules {
		if mod.Type == "script" {
			entry = mod.Entry
			break
		}
	}

	lang := "unknown"
	source := ""
	if fileExists(filepath.Join(projectPath, "src", "main.ts")) {
		lang = "typescript"
		source = "src/main.ts"
	} else if fileExists(filepath.Join(projectPath, "src", "main.js")) {
		lang = "javascript"
		source = "src/main.js"
	}

	warnings := make([]string, 0)
	if entry != "" && !fileExists(filepath.Join(projectPath, "behavior_pack", filepath.FromSlash(entry))) {
		warnings = append(warnings, fmt.Sprintf("script entrypoint %q does not exist in behavior_pack", entry))
	}

	return &InspectAddonWorkspaceOutput{
		ProjectPath:      projectPath,
		HasBehaviorPack:  hasBP,
		HasResourcePack:  hasRP,
		Language:         lang,
		Entrypoint:       entry,
		SourceEntrypoint: source,
		Modules:          modules,
		Warnings:         warnings,
	}, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

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
	BPSource         string   `json:"bp_source,omitempty"`
	RPSource         string   `json:"rp_source,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
}

func inspectAddonWorkspace(projectPath string) (*InspectAddonWorkspaceOutput, error) {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return nil, fmt.Errorf("project_path is required")
	}

	layout, err := resolvePackLayout(projectPath, "", "", "")
	if err != nil {
		return nil, err
	}
	if layout.BPSource == "" {
		return nil, fmt.Errorf("behavior pack manifest not found (checked behavior_pack/, static/bp/, src/bp/, packs/bp/, addon/bp/)")
	}

	bpManifestPath := filepath.Join(projectPath, filepath.FromSlash(layout.BPSource), "manifest.json")
	rpManifestPath := ""
	if layout.RPSource != "" {
		rpManifestPath = filepath.Join(projectPath, filepath.FromSlash(layout.RPSource), "manifest.json")
	}

	hasBP := fileExists(bpManifestPath)
	hasRP := rpManifestPath != "" && fileExists(rpManifestPath)
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
	} else if hasTSFiles(filepath.Join(projectPath, "src")) {
		lang = "typescript"
	}

	warnings := make([]string, 0)
	if entry != "" {
		entryAbs := filepath.Join(projectPath, filepath.FromSlash(layout.BPSource), filepath.FromSlash(entry))
		if !fileExists(entryAbs) {
			warnings = append(warnings, fmt.Sprintf("script entrypoint %q does not exist in %s", entry, layout.BPSource))
		}
	}

	return &InspectAddonWorkspaceOutput{
		ProjectPath:      projectPath,
		HasBehaviorPack:  hasBP,
		HasResourcePack:  hasRP,
		Language:         lang,
		Entrypoint:       entry,
		SourceEntrypoint: source,
		Modules:          modules,
		BPSource:         layout.BPSource,
		RPSource:         layout.RPSource,
		Warnings:         warnings,
	}, nil
}

func hasTSFiles(dir string) bool {
	found := false
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".ts") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

package manifestdoctor

import (
	"encoding/json"
	"fmt"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
)

func RunFixup(manifestJSON string, fixIDs []string, fixAllFixable bool, opts *DoctorOptions) *FixupOutput {
	if opts == nil {
		opts = &DoctorOptions{MinEngineVersion: []int{1, 21, 60}}
	}
	if len(opts.MinEngineVersion) != 3 {
		opts.MinEngineVersion = []int{1, 21, 60}
	}

	doctorResult := RunDoctor(manifestJSON, opts)

	toApply := make(map[string]bool)
	if fixAllFixable || len(fixIDs) == 0 {
		for _, f := range doctorResult.Fixes {
			toApply[f] = true
		}
	} else {
		for _, id := range fixIDs {
			toApply[id] = true
		}
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(manifestJSON), &raw); err != nil {
		unfixable := doctorResult.Errors
		if unfixable == nil {
			unfixable = []Finding{}
		}
		return &FixupOutput{
			OriginalManifest: manifestJSON,
			FixedManifest:    manifestJSON,
			AppliedFixes:     []FixResult{},
			UnfixableErrors:  unfixable,
			Summary:          fmt.Sprintf("Cannot parse manifest JSON: %v", err),
		}
	}

	var appliedFixes []FixResult
	var unfixableErrors []Finding
	for _, f := range doctorResult.Errors {
		if !f.Fixable {
			unfixableErrors = append(unfixableErrors, f)
		}
	}
	if unfixableErrors == nil {
		unfixableErrors = []Finding{}
	}

	if toApply["missing_format_version"] || toApply["format_version_not_2"] {
		raw["format_version"] = float64(2)
		appliedFixes = append(appliedFixes, FixResult{Rule: "missing_format_version", Note: "Set format_version to 2"})
	}

	ensureHeader := func() map[string]any {
		h, ok := raw["header"].(map[string]any)
		if !ok {
			h = make(map[string]any)
			raw["header"] = h
		}
		return h
	}

	if toApply["missing_header_name"] {
		header := ensureHeader()
		header["name"] = "My Addon"
		appliedFixes = append(appliedFixes, FixResult{Rule: "missing_header_name", Note: "Set header.name to 'My Addon'"})
	}

	if toApply["missing_header_description"] {
		header := ensureHeader()
		header["description"] = "A Minecraft Bedrock addon"
		appliedFixes = append(appliedFixes, FixResult{Rule: "missing_header_description", Note: "Set header.description to 'A Minecraft Bedrock addon'"})
	}

	if toApply["missing_header_uuid"] || toApply["invalid_header_uuid"] {
		header := ensureHeader()
		header["uuid"] = manifest.GenerateUUID()
		appliedFixes = append(appliedFixes, FixResult{Rule: "missing_header_uuid", Note: "Generated new UUID for header"})
	}

	if toApply["duplicate_uuid"] {
		seen := make(map[string]bool)
		if header, ok := raw["header"].(map[string]any); ok {
			if u, ok := header["uuid"].(string); ok {
				if seen[u] {
					header["uuid"] = manifest.GenerateUUID()
				}
				seen[u] = true
			}
		}
		if modules, ok := raw["modules"].([]any); ok {
			for _, m := range modules {
				if mod, ok := m.(map[string]any); ok {
					if u, ok := mod["uuid"].(string); ok {
						if seen[u] {
							mod["uuid"] = manifest.GenerateUUID()
						}
						seen[u] = true
					}
				}
			}
		}
		if deps, ok := raw["dependencies"].([]any); ok {
			for _, d := range deps {
				if dep, ok := d.(map[string]any); ok {
					if u, ok := dep["uuid"].(string); ok {
						if seen[u] {
							dep["uuid"] = manifest.GenerateUUID()
						}
						seen[u] = true
					}
				}
			}
		}
		appliedFixes = append(appliedFixes, FixResult{Rule: "duplicate_uuid", Note: "Regenerated duplicate UUIDs"})
	}

	if toApply["invalid_header_version"] {
		header := ensureHeader()
		header["version"] = []any{float64(1), float64(0), float64(0)}
		appliedFixes = append(appliedFixes, FixResult{Rule: "invalid_header_version", Note: "Set header.version to [1, 0, 0]"})
	}

	if toApply["invalid_module_version"] {
		if modules, ok := raw["modules"].([]any); ok {
			for _, m := range modules {
				if mod, ok := m.(map[string]any); ok {
					mod["version"] = []any{float64(1), float64(0), float64(0)}
				}
			}
		}
		appliedFixes = append(appliedFixes, FixResult{Rule: "invalid_module_version", Note: "Set module versions to [1, 0, 0]"})
	}

	if toApply["invalid_min_engine_version"] {
		header := ensureHeader()
		header["min_engine_version"] = []any{float64(opts.MinEngineVersion[0]), float64(opts.MinEngineVersion[1]), float64(opts.MinEngineVersion[2])}
		appliedFixes = append(appliedFixes, FixResult{Rule: "invalid_min_engine_version", Note: fmt.Sprintf("Set min_engine_version to %v", opts.MinEngineVersion)})
	}

	if toApply["script_module_missing_entry"] {
		if modules, ok := raw["modules"].([]any); ok {
			for _, m := range modules {
				if mod, ok := m.(map[string]any); ok {
					if t, _ := mod["type"].(string); t == "script" {
						if _, ok := mod["entry"]; !ok {
							mod["entry"] = "scripts/main.js"
						}
					}
				}
			}
		}
		appliedFixes = append(appliedFixes, FixResult{Rule: "script_module_missing_entry", Note: "Set script module entry to 'scripts/main.js'"})
	}

	if toApply["script_module_missing_language"] {
		if modules, ok := raw["modules"].([]any); ok {
			for _, m := range modules {
				if mod, ok := m.(map[string]any); ok {
					if t, _ := mod["type"].(string); t == "script" {
						if _, ok := mod["language"]; !ok {
							mod["language"] = "javascript"
						}
					}
				}
			}
		}
		appliedFixes = append(appliedFixes, FixResult{Rule: "script_module_missing_language", Note: "Set script module language to 'javascript'"})
	}

	if toApply["deprecated_module"] {
		if deps, ok := raw["dependencies"].([]any); ok {
			for _, d := range deps {
				if dep, ok := d.(map[string]any); ok {
					if name, ok := dep["module_name"].(string); ok {
						if replacement, found := deprecatedModuleMap[name]; found {
							dep["module_name"] = replacement
						}
					}
				}
			}
		}
		appliedFixes = append(appliedFixes, FixResult{Rule: "deprecated_module", Note: "Replaced deprecated mojang-* modules with @minecraft/* equivalents"})
	}

	if toApply["duplicate_dependency"] {
		if deps, ok := raw["dependencies"].([]any); ok {
			seenModule := make(map[string]bool)
			seenUUID := make(map[string]bool)
			var deduped []any
			for _, d := range deps {
				if dep, ok := d.(map[string]any); ok {
					modName, _ := dep["module_name"].(string)
					uuid, _ := dep["uuid"].(string)
					if modName != "" && seenModule[modName] {
						continue
					}
					if uuid != "" && seenUUID[uuid] {
						continue
					}
					if modName != "" {
						seenModule[modName] = true
					}
					if uuid != "" {
						seenUUID[uuid] = true
					}
					deduped = append(deduped, d)
				} else {
					deduped = append(deduped, d)
				}
			}
			raw["dependencies"] = deduped
		}
		appliedFixes = append(appliedFixes, FixResult{Rule: "duplicate_dependency", Note: "Deduplicated dependencies by module_name and uuid"})
	}

	if toApply["missing_minecraft_server"] {
		deps, ok := raw["dependencies"].([]any)
		if !ok {
			deps = []any{}
		}
		raw["dependencies"] = append(deps, map[string]any{
			"module_name": "@minecraft/server",
			"version":     "latest",
		})
		appliedFixes = append(appliedFixes, FixResult{Rule: "missing_minecraft_server", Note: "Added @minecraft/server dependency"})
	}

	fixedBytes, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return &FixupOutput{
			OriginalManifest: manifestJSON,
			FixedManifest:    manifestJSON,
			AppliedFixes:     appliedFixes,
			UnfixableErrors:  unfixableErrors,
			Summary:          fmt.Sprintf("Applied %d fix(es), failed to marshal fixed JSON: %v", len(appliedFixes), err),
		}
	}

	summary := fmt.Sprintf("Applied %d fix(es)", len(appliedFixes))
	if len(unfixableErrors) > 0 {
		summary += fmt.Sprintf(", %d unfixable error(s)", len(unfixableErrors))
	}

	return &FixupOutput{
		OriginalManifest: manifestJSON,
		FixedManifest:    string(fixedBytes),
		AppliedFixes:     appliedFixes,
		UnfixableErrors:  unfixableErrors,
		Summary:          summary,
	}
}

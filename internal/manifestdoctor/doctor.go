package manifestdoctor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/models"
)

func RunDoctor(manifestJSON string, opts *DoctorOptions) *DoctorOutput {
	if opts == nil {
		opts = &DoctorOptions{MinEngineVersion: []int{1, 21, 60}}
	}
	if len(opts.MinEngineVersion) != 3 {
		opts.MinEngineVersion = []int{1, 21, 60}
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(manifestJSON), &raw); err != nil {
		return &DoctorOutput{
			OK:      false,
			Errors:  []Finding{{Rule: "invalid_json", Severity: "error", Message: fmt.Sprintf("Invalid JSON: %v", err), Path: "", Fixable: false}},
			Summary: "1 error(s)",
		}
	}

	var parsed *models.Manifest
	if p, err := manifest.ParseManifest(manifestJSON); err == nil {
		parsed = &p
	}

	packKind := detectPackKind(raw)

	var allFindings []Finding
	for _, check := range AllRuleChecks() {
		allFindings = append(allFindings, check(raw, parsed, opts)...)
	}

	var errors, warnings []Finding
	var fixes []string
	seenFixes := make(map[string]bool)
	for _, f := range allFindings {
		if f.Severity == "error" {
			errors = append(errors, f)
		} else {
			warnings = append(warnings, f)
		}
		if f.Fixable && !seenFixes[f.Rule] {
			fixes = append(fixes, f.Rule)
			seenFixes[f.Rule] = true
		}
	}

	sort.Slice(errors, func(i, j int) bool { return errors[i].Path < errors[j].Path })
	sort.Slice(warnings, func(i, j int) bool { return warnings[i].Path < warnings[j].Path })
	sort.Strings(fixes)

	var parts []string
	if len(errors) > 0 {
		parts = append(parts, fmt.Sprintf("%d error(s)", len(errors)))
	}
	if len(warnings) > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s)", len(warnings)))
	}
	summary := "No issues found"
	if len(parts) > 0 {
		summary = strings.Join(parts, ", ")
	}

	return &DoctorOutput{
		OK:               len(errors) == 0,
		DetectedPackKind: packKind,
		Errors:           errors,
		Warnings:         warnings,
		Fixes:            fixes,
		Summary:          summary,
	}
}

func detectPackKind(raw map[string]any) string {
	modules, ok := raw["modules"].([]any)
	if !ok || len(modules) == 0 {
		return "unknown"
	}
	hasScript := false
	hasResources := false
	for _, m := range modules {
		mod, ok := m.(map[string]any)
		if !ok {
			continue
		}
		t, _ := mod["type"].(string)
		if t == "script" {
			hasScript = true
		}
		if t == "resources" {
			hasResources = true
		}
	}
	if hasScript {
		return "behavior"
	}
	if hasResources {
		return "resource"
	}
	return "unknown"
}

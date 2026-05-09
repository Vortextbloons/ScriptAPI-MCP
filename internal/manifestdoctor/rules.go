package manifestdoctor

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/models"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

var deprecatedModuleMap = map[string]string{
	"mojang-minecraft":               "@minecraft/server",
	"mojang-minecraft-ui":            "@minecraft/server-ui",
	"mojang-minecraft-server-admin":  "@minecraft/server-admin",
	"mojang-gametest":                "@minecraft/server-gametest",
}

type RuleCheck func(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding

func AllRuleChecks() []RuleCheck {
	return []RuleCheck{
		checkInvalidJSON,
		checkMissingFormatVersion,
		checkFormatVersionNot2,
		checkMissingHeader,
		checkMissingHeaderName,
		checkMissingHeaderDescription,
		checkMissingHeaderUUID,
		checkInvalidHeaderUUID,
		checkDuplicateUUIDs,
		checkInvalidHeaderVersion,
		checkInvalidModuleVersion,
		checkInvalidMinEngineVersion,
		checkMultipleScriptModules,
		checkScriptModuleMissingEntry,
		checkScriptModuleMissingLanguage,
		checkDeprecatedModule,
		checkUnknownModule,
		checkDuplicateDependency,
		checkMissingMinecraftServer,
		checkInvalidPackDependencyVersion,
		checkNodeModulesMismatch,
	}
}

func isValidVersionArray(v any) bool {
	arr, ok := v.([]any)
	if !ok || len(arr) != 3 {
		return false
	}
	for _, elem := range arr {
		switch elem.(type) {
		case float64:
		default:
			return false
		}
	}
	return true
}

func collectUUIDs(raw map[string]any) []string {
	var uuids []string
	if header, ok := raw["header"].(map[string]any); ok {
		if u, ok := header["uuid"].(string); ok {
			uuids = append(uuids, u)
		}
	}
	if modules, ok := raw["modules"].([]any); ok {
		for _, m := range modules {
			if mod, ok := m.(map[string]any); ok {
				if u, ok := mod["uuid"].(string); ok {
					uuids = append(uuids, u)
				}
			}
		}
	}
	if deps, ok := raw["dependencies"].([]any); ok {
		for _, d := range deps {
			if dep, ok := d.(map[string]any); ok {
				if u, ok := dep["uuid"].(string); ok {
					uuids = append(uuids, u)
				}
			}
		}
	}
	return uuids
}

func checkInvalidJSON(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	return nil
}

func checkMissingFormatVersion(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	if _, ok := raw["format_version"]; !ok {
		return []Finding{
			{Rule: "missing_format_version", Severity: "error", Message: "Missing 'format_version' field", Path: "format_version", Fixable: true},
		}
	}
	return nil
}

func checkFormatVersionNot2(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	v, ok := raw["format_version"]
	if !ok {
		return nil
	}
	if f, ok := v.(float64); ok && f == 2 {
		return nil
	}
	return []Finding{
		{Rule: "format_version_not_2", Severity: "error", Message: "'format_version' should be 2", Path: "format_version", Fixable: true},
	}
}

func checkMissingHeader(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	h, ok := raw["header"]
	if !ok {
		return []Finding{
			{Rule: "missing_header", Severity: "error", Message: "Missing 'header' section", Path: "header", Fixable: false},
		}
	}
	if _, ok := h.(map[string]any); !ok {
		return []Finding{
			{Rule: "missing_header", Severity: "error", Message: "'header' is not a valid object", Path: "header", Fixable: false},
		}
	}
	return nil
}

func checkMissingHeaderName(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	h, ok := raw["header"].(map[string]any)
	if !ok {
		return nil
	}
	if _, ok := h["name"]; !ok {
		return []Finding{
			{Rule: "missing_header_name", Severity: "error", Message: "Missing 'header.name' field", Path: "header.name", Fixable: true},
		}
	}
	return nil
}

func checkMissingHeaderDescription(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	h, ok := raw["header"].(map[string]any)
	if !ok {
		return nil
	}
	if _, ok := h["description"]; !ok {
		return []Finding{
			{Rule: "missing_header_description", Severity: "warning", Message: "Missing 'header.description' field", Path: "header.description", Fixable: true},
		}
	}
	return nil
}

func checkMissingHeaderUUID(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	h, ok := raw["header"].(map[string]any)
	if !ok {
		return nil
	}
	if _, ok := h["uuid"]; !ok {
		return []Finding{
			{Rule: "missing_header_uuid", Severity: "error", Message: "Missing 'header.uuid' field", Path: "header.uuid", Fixable: true},
		}
	}
	return nil
}

func checkInvalidHeaderUUID(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	h, ok := raw["header"].(map[string]any)
	if !ok {
		return nil
	}
	u, ok := h["uuid"].(string)
	if !ok {
		return nil
	}
	if !uuidRegex.MatchString(u) {
		return []Finding{
			{Rule: "invalid_header_uuid", Severity: "error", Message: "Invalid 'header.uuid' format: not a valid v4 UUID", Path: "header.uuid", Fixable: true},
		}
	}
	return nil
}

func checkDuplicateUUIDs(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	uuids := collectUUIDs(raw)
	seen := make(map[string]int)
	for _, u := range uuids {
		seen[u]++
	}
	var duplicates []string
	for u, count := range seen {
		if count > 1 {
			duplicates = append(duplicates, u)
		}
	}
	if len(duplicates) > 0 {
		return []Finding{
			{Rule: "duplicate_uuid", Severity: "error", Message: fmt.Sprintf("Duplicate UUID(s) found: %s", strings.Join(duplicates, ", ")), Path: "", Fixable: true},
		}
	}
	return nil
}

func checkInvalidHeaderVersion(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	h, ok := raw["header"].(map[string]any)
	if !ok {
		return nil
	}
	v, ok := h["version"]
	if !ok {
		return nil
	}
	if !isValidVersionArray(v) {
		return []Finding{
			{Rule: "invalid_header_version", Severity: "error", Message: "Invalid 'header.version' format (expected [major, minor, patch])", Path: "header.version", Fixable: true},
		}
	}
	return nil
}

func checkInvalidModuleVersion(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	modules, ok := raw["modules"].([]any)
	if !ok {
		return nil
	}
	var findings []Finding
	for i, m := range modules {
		mod, ok := m.(map[string]any)
		if !ok {
			continue
		}
		v, ok := mod["version"]
		if !ok {
			findings = append(findings, Finding{
				Rule: "invalid_module_version", Severity: "error", Message: fmt.Sprintf("Module at index %d is missing version", i), Path: fmt.Sprintf("modules[%d].version", i), Fixable: true,
			})
			continue
		}
		if !isValidVersionArray(v) {
			findings = append(findings, Finding{
				Rule: "invalid_module_version", Severity: "error", Message: fmt.Sprintf("Module at index %d has invalid version format", i), Path: fmt.Sprintf("modules[%d].version", i), Fixable: true,
			})
		}
	}
	return findings
}

func checkInvalidMinEngineVersion(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	h, ok := raw["header"].(map[string]any)
	if !ok {
		return nil
	}
	v, ok := h["min_engine_version"]
	if !ok {
		return nil
	}
	if !isValidVersionArray(v) {
		return []Finding{
			{Rule: "invalid_min_engine_version", Severity: "warning", Message: "Invalid 'header.min_engine_version' format (expected [major, minor, patch])", Path: "header.min_engine_version", Fixable: true},
		}
	}
	return nil
}

func checkMultipleScriptModules(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	modules, ok := raw["modules"].([]any)
	if !ok {
		return nil
	}
	count := 0
	for _, m := range modules {
		mod, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := mod["type"].(string); t == "script" {
			count++
		}
	}
	if count > 1 {
		return []Finding{
			{Rule: "multiple_script_modules", Severity: "error", Message: fmt.Sprintf("Found %d script modules; only one script module is allowed", count), Path: "modules", Fixable: false},
		}
	}
	return nil
}

func checkScriptModuleMissingEntry(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	modules, ok := raw["modules"].([]any)
	if !ok {
		return nil
	}
	var findings []Finding
	for i, m := range modules {
		mod, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := mod["type"].(string); t == "script" {
			if _, ok := mod["entry"]; !ok {
				findings = append(findings, Finding{
					Rule: "script_module_missing_entry", Severity: "error", Message: fmt.Sprintf("Script module at index %d is missing 'entry' field", i), Path: fmt.Sprintf("modules[%d].entry", i), Fixable: true,
				})
			}
		}
	}
	return findings
}

func checkScriptModuleMissingLanguage(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	modules, ok := raw["modules"].([]any)
	if !ok {
		return nil
	}
	var findings []Finding
	for i, m := range modules {
		mod, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := mod["type"].(string); t == "script" {
			if _, ok := mod["language"]; !ok {
				findings = append(findings, Finding{
					Rule: "script_module_missing_language", Severity: "error", Message: fmt.Sprintf("Script module at index %d is missing 'language' field", i), Path: fmt.Sprintf("modules[%d].language", i), Fixable: true,
				})
			}
		}
	}
	return findings
}

func checkDeprecatedModule(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	if parsed == nil {
		return nil
	}
	var findings []Finding
	for i, d := range parsed.Dependencies {
		if replacement, ok := deprecatedModuleMap[d.ModuleName]; ok {
			findings = append(findings, Finding{
				Rule: "deprecated_module", Severity: "error", Message: fmt.Sprintf("Deprecated module '%s' — use '%s' instead", d.ModuleName, replacement), Path: fmt.Sprintf("dependencies[%d].module_name", i), Fixable: true,
			})
		}
	}
	return findings
}

func checkUnknownModule(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	if parsed == nil {
		return nil
	}
	var findings []Finding
	for i, d := range parsed.Dependencies {
		if !strings.HasPrefix(d.ModuleName, "@minecraft/") {
			continue
		}
		isAllowed := false
		for _, a := range models.AllowedModules {
			if d.ModuleName == a {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			findings = append(findings, Finding{
				Rule: "unknown_module", Severity: "warning", Message: fmt.Sprintf("Unknown module '%s' is not in the allowed modules list", d.ModuleName), Path: fmt.Sprintf("dependencies[%d].module_name", i), Fixable: false,
			})
		}
	}
	return findings
}

func checkDuplicateDependency(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	if parsed == nil {
		return nil
	}
	seenModule := make(map[string]bool)
	seenUUID := make(map[string]bool)
	var findings []Finding
	for i, d := range parsed.Dependencies {
		if d.ModuleName != "" {
			if seenModule[d.ModuleName] {
				findings = append(findings, Finding{
					Rule: "duplicate_dependency", Severity: "warning", Message: fmt.Sprintf("Duplicate dependency by module_name '%s' at index %d", d.ModuleName, i), Path: fmt.Sprintf("dependencies[%d]", i), Fixable: true,
				})
			}
			seenModule[d.ModuleName] = true
		}
		if d.UUID != "" {
			if seenUUID[d.UUID] {
				findings = append(findings, Finding{
					Rule: "duplicate_dependency", Severity: "warning", Message: fmt.Sprintf("Duplicate dependency by UUID '%s' at index %d", d.UUID, i), Path: fmt.Sprintf("dependencies[%d]", i), Fixable: true,
				})
			}
			seenUUID[d.UUID] = true
		}
	}
	return findings
}

func checkMissingMinecraftServer(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	if parsed == nil {
		return nil
	}
	hasScript := false
	for _, m := range parsed.Modules {
		if m.Type == "script" {
			hasScript = true
			break
		}
	}
	if !hasScript {
		return nil
	}
	for _, d := range parsed.Dependencies {
		if d.ModuleName == "@minecraft/server" {
			return nil
		}
	}
	return []Finding{
		{Rule: "missing_minecraft_server", Severity: "error", Message: "Script module requires '@minecraft/server' dependency", Path: "dependencies", Fixable: true},
	}
}

func checkInvalidPackDependencyVersion(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	if parsed == nil {
		return nil
	}
	var findings []Finding
	for i, d := range parsed.Dependencies {
		if d.ModuleName == "" && d.UUID != "" {
			if d.Version == "" {
				findings = append(findings, Finding{
					Rule: "invalid_pack_dependency_version", Severity: "warning", Message: fmt.Sprintf("UUID-based dependency at index %d has empty version", i), Path: fmt.Sprintf("dependencies[%d].version", i), Fixable: false,
				})
			}
		}
	}
	return findings
}

func checkNodeModulesMismatch(raw map[string]any, parsed *models.Manifest, opts *DoctorOptions) []Finding {
	if !opts.CheckLocalModules || opts.ProjectPath == "" || parsed == nil {
		return nil
	}
	manifestVersions := make(map[string]string)
	for _, d := range parsed.Dependencies {
		if d.ModuleName != "" {
			manifestVersions[d.ModuleName] = d.Version
		}
	}
	if len(manifestVersions) == 0 {
		return nil
	}
	warnings, err := npm.ValidateInstalledModules(opts.ProjectPath, manifestVersions)
	if err != nil {
		return []Finding{
			{Rule: "node_modules_mismatch", Severity: "warning", Message: fmt.Sprintf("Failed to validate node_modules: %v", err), Path: "", Fixable: false},
		}
	}
	var findings []Finding
	for _, w := range warnings {
		findings = append(findings, Finding{
			Rule: "node_modules_mismatch", Severity: "warning", Message: w, Path: "", Fixable: false,
		})
	}
	return findings
}

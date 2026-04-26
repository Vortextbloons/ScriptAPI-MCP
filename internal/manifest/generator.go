package manifest

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/models"
)

// GenerateUUID creates a new v4 UUID string
func GenerateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to a deterministic UUID on failure
		return "00000000-0000-0000-0000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant is 10
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// GenerateBP creates a Behavior Pack manifest
func GenerateBP(name, description string, deps []models.Dependency, bpUUID string) models.Manifest {
	return models.Manifest{
		FormatVersion: 2,
		Header: models.ManifestHeader{
			Name:        name,
			Description: description,
			UUID:        bpUUID,
			Version:     []int{1, 0, 0},
			MinEngineVersion: []int{1, 21, 60},
		},
		Modules: []models.ManifestModule{
			{
				Type:    "data",
				UUID:    GenerateUUID(),
				Version: []int{1, 0, 0},
			},
			{
				Type:     "script",
				UUID:     GenerateUUID(),
				Version:  []int{1, 0, 0},
				Language: "javascript",
				Entry:    "scripts/main.js",
			},
		},
		Dependencies: deps,
	}
}

// GenerateRP creates a Resource Pack manifest linked to a BP
func GenerateRP(name, description, rpUUID, bpUUID string) models.Manifest {
	return models.Manifest{
		FormatVersion: 2,
		Header: models.ManifestHeader{
			Name:        name + " RP",
			Description: description,
			UUID:        rpUUID,
			Version:     []int{1, 0, 0},
			MinEngineVersion: []int{1, 21, 60},
		},
		Modules: []models.ManifestModule{
			{
				Type:    "resources",
				UUID:    GenerateUUID(),
				Version: []int{1, 0, 0},
			},
		},
		Dependencies: []models.Dependency{
			{UUID: bpUUID, Version: "1.0.0"},
		},
	}
}

// GenerateStarterCode returns minimal starter JS/TS
func GenerateStarterCode(lang, version string) map[string]string {
	files := make(map[string]string)
	
	jsCode := fmt.Sprintf(`import { world, system } from "@minecraft/server";

console.warn("Script loaded for version %s");
`, version)

	if lang == "typescript" {
		files["scripts/main.ts"] = jsCode + `
// TypeScript entry point
`
		files["tsconfig.json"] = generateTSConfig()
	} else {
		files["scripts/main.js"] = jsCode
	}
	return files
}

func generateTSConfig() string {
	cfg := map[string]interface{}{
		"compilerOptions": map[string]interface{}{
			"target":           "ES2020",
			"module":           "ES2020",
			"moduleResolution": "node",
			"strict":           true,
			"esModuleInterop":  true,
			"skipLibCheck":     true,
			"forceConsistentCasingInFileNames": true,
			"lib":              []string{"ES2020"},
			"types":            []string{},
		},
		"include": []string{"scripts/**/*"},
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return string(b)
}

// BuildDependencies creates the dependency list for a BP manifest
func BuildDependencies(serverVersion string, needsUI bool) []models.Dependency {
	deps := []models.Dependency{
		{ModuleName: "@minecraft/server", Version: serverVersion},
	}
	if needsUI {
		deps = append(deps, models.Dependency{ModuleName: "@minecraft/server-ui", Version: serverVersion})
	}
	return deps
}

// UpdateDependencies modifies a manifest's dependency list
func UpdateDependencies(manifest *models.Manifest, added, removed []string) error {
	if err := ValidateChanges(added, removed); err != nil {
		return err
	}

	// Build set of current dependencies by module name
	depMap := make(map[string]models.Dependency)
	for _, d := range manifest.Dependencies {
		if d.ModuleName != "" {
			depMap[d.ModuleName] = d
		}
	}

	// Remove
	for _, mod := range removed {
		delete(depMap, mod)
	}

	// Add
	for _, mod := range added {
		depMap[mod] = models.Dependency{ModuleName: mod, Version: "latest"}
	}

	// Rebuild slice
	newDeps := make([]models.Dependency, 0, len(depMap))
	for _, d := range depMap {
		newDeps = append(newDeps, d)
	}
	manifest.Dependencies = newDeps
	return nil
}

// FormatManifest serializes a manifest to a pretty JSON string
func FormatManifest(m models.Manifest) (string, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ParseManifest parses a manifest JSON string
func ParseManifest(s string) (models.Manifest, error) {
	var m models.Manifest
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return m, fmt.Errorf("invalid manifest JSON: %w", err)
	}
	return m, nil
}

// FileStructure returns the recommended folder layout
func FileStructure(addonName string, needsRP bool, lang string) []string {
	base := []string{
		addonName + "/",
		addonName + "/behavior_pack/",
		addonName + "/behavior_pack/manifest.json",
		addonName + "/behavior_pack/scripts/",
		addonName + "/behavior_pack/scripts/main." + lang,
	}
	if needsRP {
		base = append(base,
			addonName+"/resource_pack/",
			addonName+"/resource_pack/manifest.json",
		)
	}
	if lang == "typescript" {
		base = append(base, addonName+"/behavior_pack/tsconfig.json")
	}
	return base
}

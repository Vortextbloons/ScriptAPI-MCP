package manifest

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/models"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
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
	sortedDeps := append([]models.Dependency(nil), deps...)
	sort.Slice(sortedDeps, func(i, j int) bool {
		if sortedDeps[i].ModuleName == sortedDeps[j].ModuleName {
			return sortedDeps[i].Version < sortedDeps[j].Version
		}
		return sortedDeps[i].ModuleName < sortedDeps[j].ModuleName
	})

	return models.Manifest{
		FormatVersion: 2,
		Header: models.ManifestHeader{
			Name:             name,
			Description:      description,
			UUID:             bpUUID,
			Version:          []int{1, 0, 0},
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
		Dependencies: sortedDeps,
	}
}

// GenerateRP creates a Resource Pack manifest linked to a BP
func GenerateRP(name, description, rpUUID, bpUUID string) models.Manifest {
	return models.Manifest{
		FormatVersion: 2,
		Header: models.ManifestHeader{
			Name:             name + " RP",
			Description:      description,
			UUID:             rpUUID,
			Version:          []int{1, 0, 0},
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
		files["src/main.ts"] = jsCode + `
// TypeScript entry point
`
		files["tsconfig.json"] = generateTSConfig()
	} else {
		files["src/main.js"] = jsCode
	}

	// Keep output order predictable for callers that iterate over the map keys.
	ordered := make(map[string]string, len(files))
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		ordered[k] = files[k]
	}
	return ordered
}

func generateTSConfig() string {
	cfg := map[string]interface{}{
		"compilerOptions": map[string]interface{}{
			"target":                           "ES2020",
			"module":                           "ES2020",
			"moduleResolution":                 "node",
			"strict":                           true,
			"esModuleInterop":                  true,
			"skipLibCheck":                     true,
			"forceConsistentCasingInFileNames": true,
			"lib":                              []string{"ES2020"},
			"types":                            []string{},
		},
		"include": []string{"src/**/*"},
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return string(b)
}

// GeneratePackageJSON creates a package.json with esbuild build scripts and resolved dependencies
func GeneratePackageJSON(addonName string, deps []models.Dependency, lang string) string {
	depMap := make(map[string]string)
	var externalFlags []string
	for _, dep := range deps {
		if dep.ModuleName != "" {
			npmVer := dep.Version
			if !strings.HasPrefix(npmVer, "^") && !strings.HasPrefix(npmVer, "~") {
				npmVer = "^" + npmVer
			}
			depMap[dep.ModuleName] = npmVer
			externalFlags = append(externalFlags, "--external:"+dep.ModuleName)
		}
	}

	scripts := make(map[string]string)
	devDeps := make(map[string]string)

	if lang == "typescript" {
		externalStr := strings.Join(externalFlags, " ")
		scripts["build"] = fmt.Sprintf("esbuild src/main.ts --bundle --format=esm --target=es2020 --outfile=behavior_pack/scripts/main.js %s && node scripts/deploy.js dev", externalStr)
		scripts["typecheck"] = "tsc --noEmit"
		devDeps["esbuild"] = "^0.25.9"
		devDeps["typescript"] = "^5.9.2"
	} else {
		scripts["build"] = "node scripts/deploy.js dev"
		devDeps["esbuild"] = "^0.25.9"
	}
	scripts["deploy:dev"] = "node scripts/deploy.js dev"
	scripts["deploy:prod"] = "node scripts/deploy.js prod"

	pkg := map[string]interface{}{
		"name":    strings.ToLower(addonName),
		"private": true,
		"scripts": scripts,
	}

	if len(devDeps) > 0 {
		pkg["devDependencies"] = devDeps
	}
	if len(depMap) > 0 {
		pkg["dependencies"] = depMap
	}

	b, _ := json.MarshalIndent(pkg, "", "  ")
	return string(b) + "\n"
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

// BuildDependenciesWithChannel creates the dependency list for a BP manifest by resolving versions from actual npm data.
// modules is a list of module names (e.g., ["@minecraft/server", "@minecraft/server-ui"])
// minecraftVersion is the Minecraft version (e.g., "1.21.60" or "latest")
// channel is "stable" or "beta"
func BuildDependenciesWithChannel(npmClient *npm.Client, modules []string, minecraftVersion string, channel string) ([]models.Dependency, error) {
	if len(modules) == 0 {
		// Default to server module
		modules = []string{"@minecraft/server"}
	}

	if channel == "" {
		channel = "beta"
	}

	// Fetch version matrices for all modules
	versionMatrices := make(map[string]*npm.VersionMatrix)
	for _, mod := range modules {
		vm, err := npmClient.FetchVersionMatrix(mod)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch versions for %s: %w", mod, err)
		}
		versionMatrices[mod] = vm
	}

	// Resolve versions for each module
	resolvedVersions, err := npm.ResolveModuleVersions(versionMatrices, minecraftVersion, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve versions: %w", err)
	}

	// Build dependency list
	deps := make([]models.Dependency, 0, len(resolvedVersions))
	for _, mod := range modules {
		if ver, ok := resolvedVersions[mod]; ok {
			deps = append(deps, models.Dependency{
				ModuleName: mod,
				Version:    ver,
			})
		}
	}

	return deps, nil
}

// BuildDependenciesWithValidation resolves versions and validates against installed node_modules
func BuildDependenciesWithValidation(projectPath string, npmClient *npm.Client, modules []string, minecraftVersion string, channel string) ([]models.Dependency, []string, error) {
	deps, err := BuildDependenciesWithChannel(npmClient, modules, minecraftVersion, channel)
	if err != nil {
		return nil, nil, err
	}

	// Build version map for validation
	manifestVersions := make(map[string]string)
	for _, dep := range deps {
		manifestVersions[dep.ModuleName] = dep.Version
	}

	// Validate against installed modules
	warnings, err := npm.ValidateInstalledModules(projectPath, manifestVersions)
	if err != nil {
		return deps, warnings, err
	}

	return deps, warnings, nil
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
func FileStructure(addonName string, needsRP bool, lang string, createDeploy bool) []string {
	ext := "js"
	if lang == "typescript" {
		ext = "ts"
	}
	base := []string{
		"behavior_pack/",
		"behavior_pack/manifest.json",
		"behavior_pack/pack_icon.png",
		"behavior_pack/scripts/",
		"src/",
		"src/main." + ext,
	}
	if needsRP {
		base = append(base,
			"resource_pack/",
			"resource_pack/manifest.json",
			"resource_pack/pack_icon.png",
		)
	}
	if lang == "typescript" {
		base = append(base, "tsconfig.json")
	}
	if createDeploy {
		base = append(base,
			"package.json",
			"scripts/",
			"scripts/deploy.js",
		)
	}
	sort.Strings(base)
	return base
}

package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PackageJSON represents a package.json file structure
type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Dependencies    map[string]string `json:"dependencies,omitempty"`
	DevDependencies map[string]string `json:"devDependencies,omitempty"`
}

// InstalledModule represents a module found in node_modules
type InstalledModule struct {
	Name          string
	Version       string
	IsBeta        bool
	ManifestPath  string
	PackagePath   string
}

// ValidateInstalledModules checks if the resolved manifest versions match what's installed in node_modules
func ValidateInstalledModules(projectPath string, manifestVersions map[string]string) ([]string, error) {
	warnings := make([]string, 0)
	projectPath = NormalizeProjectPath(projectPath)

	moduleNames := make([]string, 0, len(manifestVersions))
	for moduleName := range manifestVersions {
		moduleNames = append(moduleNames, moduleName)
	}
	sort.Strings(moduleNames)

	for _, moduleName := range moduleNames {
		manifestVersion := manifestVersions[moduleName]
		installed, err := GetInstalledModule(projectPath, moduleName)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("WARNING: %s not found in node_modules: %v", moduleName, err))
			continue
		}

		// Normalize both versions for comparison
		normalizedManifest := NormalizeVersion(manifestVersion)
		normalizedInstalled := NormalizeVersion(installed.Version)

		if normalizedManifest != normalizedInstalled {
			warnings = append(warnings, fmt.Sprintf(
				"WARNING: %s version mismatch - manifest: %s, installed: %s",
				moduleName, normalizedManifest, normalizedInstalled,
			))
		}
	}

	return warnings, nil
}

// NormalizeProjectPath returns a stable validation path.
func NormalizeProjectPath(projectPath string) string {
	if strings.TrimSpace(projectPath) == "" {
		return "."
	}
	return projectPath
}

// GetInstalledModule reads a module's package.json from node_modules
func GetInstalledModule(projectPath string, moduleName string) (*InstalledModule, error) {
	// Convert @minecraft/server to @minecraft/server
	modulePath := filepath.Join(projectPath, "node_modules", moduleName)
	packagePath := filepath.Join(modulePath, "package.json")

	data, err := os.ReadFile(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", packagePath, err)
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", packagePath, err)
	}

	return &InstalledModule{
		Name:         moduleName,
		Version:      pkg.Version,
		IsBeta:       strings.Contains(pkg.Version, "beta"),
		ManifestPath: packagePath,
		PackagePath:  modulePath,
	}, nil
}

// GetInstalledMinecraftModules scans node_modules for all @minecraft/* modules.
func GetInstalledMinecraftModules(projectPath string) ([]InstalledModule, error) {
	projectPath = NormalizeProjectPath(projectPath)
	nodeModulesPath := filepath.Join(projectPath, "node_modules", "@minecraft")

	entries, err := os.ReadDir(nodeModulesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []InstalledModule{}, nil
		}
		return nil, fmt.Errorf("failed to read @minecraft modules: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	modules := make([]InstalledModule, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		moduleName := "@minecraft/" + entry.Name()
		installed, err := GetInstalledModule(projectPath, moduleName)
		if err != nil {
			continue
		}
		modules = append(modules, *installed)
	}

	return modules, nil
}

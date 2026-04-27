package tools

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

func validateInstalledDependencies(projectPath string, deps map[string]string) []string {
	projectPath = normalizeProjectPath(projectPath)
	if projectPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			projectPath = cwd
		} else {
			return []string{fmt.Sprintf("WARNING: local node_modules validation failed: %v", err)}
		}
	}

	warnings, err := npm.ValidateInstalledModules(projectPath, deps)
	if err != nil {
		return []string{fmt.Sprintf("WARNING: local node_modules validation failed: %v", err)}
	}
	if len(warnings) == 0 {
		return nil
	}

	sort.Strings(warnings)
	return warnings
}

func normalizeProjectPath(projectPath string) string {
	projectPath = strings.TrimSpace(projectPath)
	if projectPath == "" {
		return ""
	}
	return projectPath
}

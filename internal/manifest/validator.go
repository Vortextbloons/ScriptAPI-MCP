package manifest

import (
	"fmt"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/models"
)

// ValidateChanges checks that added/removed modules are valid
func ValidateChanges(added, removed []string) error {
	for _, mod := range added {
		if IsDeprecated(mod) {
			return fmt.Errorf("module %q is deprecated and cannot be added. Use @minecraft/server instead.", mod)
		}
		if !IsAllowed(mod) {
			return fmt.Errorf("module %q is not an allowed Bedrock Script API module", mod)
		}
	}
	for _, mod := range removed {
		if IsDeprecated(mod) {
			return fmt.Errorf("module %q is deprecated and cannot be referenced", mod)
		}
	}
	return nil
}

// IsAllowed checks if a module is in the whitelist
func IsAllowed(mod string) bool {
	for _, a := range models.AllowedModules {
		if a == mod {
			return true
		}
	}
	return false
}

// IsDeprecated checks if a module is explicitly forbidden
func IsDeprecated(mod string) bool {
	for _, d := range models.DeprecatedModules {
		if d == mod {
			return true
		}
	}
	return false
}

package npm

import (
	"fmt"
	"sort"
	"strings"
)

// LookupExactVersion checks whether a given version string exists as an exact publish for a module.
// Returns true if the version is found in the version matrix.
func (c *Client) LookupExactVersion(module, version string) (bool, error) {
	vm, err := c.FetchVersionMatrix(module)
	if err != nil {
		return false, fmt.Errorf("failed to fetch version matrix for %s: %w", module, err)
	}
	for _, v := range vm.Versions {
		if v == version {
			return true, nil
		}
	}
	return false, nil
}

// ListConcreteVersions fetches all exact versions for a module that match the given shorthand prefix.
// Returns versions sorted descending by semver (highest first).
// If no matches are found, returns an empty slice.
func (c *Client) ListConcreteVersions(module, shorthand string) ([]string, error) {
	vm, err := c.FetchVersionMatrix(module)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version matrix for %s: %w", module, err)
	}

	candidates := make([]string, 0)
	for _, v := range vm.Versions {
		if strings.HasPrefix(v, shorthand) {
			candidates = append(candidates, v)
		}
	}

	if len(candidates) == 0 {
		return candidates, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return compareSemver(candidates[i], candidates[j]) > 0
	})

	return candidates, nil
}

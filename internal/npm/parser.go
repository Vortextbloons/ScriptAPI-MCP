package npm

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ResolveModuleVersions resolves versions for multiple modules given a Minecraft version and channel.
// channel should be "stable", "beta", or "preview" (defaults to "beta" if empty).
func ResolveModuleVersions(clients map[string]*VersionMatrix, minecraftVersion string, channel string) (map[string]string, error) {
	if channel == "" {
		channel = "beta"
	}

	resolved := make(map[string]string)

	for module, vm := range clients {
		ver, err := ResolveVersionForChannel(vm, minecraftVersion, channel)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve version for %s: %w", module, err)
		}
		resolved[module] = ver
	}

	return resolved, nil
}

// NormalizeVersion converts a full npm version to the short form used in manifest.json.
// e.g., "2.7.0-beta.1.26.14-stable" -> "2.7.0-beta"
// e.g., "2.6.0-stable" -> "2.6.0"
func NormalizeVersion(version string) string {
	// Split on "-" to get base version and pre-release parts
	parts := strings.SplitN(version, "-", 2)
	base := parts[0]

	if len(parts) < 2 {
		return base
	}

	prerelease := parts[1]
	// Check if it's a beta version (contains "beta" in prerelease)
	if strings.Contains(prerelease, "beta") {
		return base + "-beta"
	}

	// For stable/other versions, just return the base
	return base
}

// ResolveVersionForChannel resolves a Minecraft version to an npm version for a specific channel.
// Channels: "stable" (no beta/preview/rc), "beta" (has -beta but NOT -preview), "preview" (has -preview).
func ResolveVersionForChannel(vm *VersionMatrix, minecraftVersion string, channel string) (string, error) {
	exact, err := ResolveExactVersionForChannel(vm, minecraftVersion, channel)
	if err != nil {
		return "", err
	}
	return NormalizeVersion(exact), nil
}

// ResolveExactVersionForChannel resolves to an exact npm publish version for a channel.
func ResolveExactVersionForChannel(vm *VersionMatrix, minecraftVersion string, channel string) (string, error) {
	if channel == "" {
		channel = "beta"
	}

	if minecraftVersion == "latest" {
		switch channel {
		case "stable":
			return resolveLatestStable(vm)
		case "beta":
			return resolveLatestBeta(vm)
		case "preview":
			return resolveLatestPreview(vm)
		default:
			return resolveLatestBeta(vm)
		}
	}

	// Clean input: remove leading "v" if present
	mcVer := strings.TrimPrefix(minecraftVersion, "v")

	// Filter versions that match the minecraft version and channel
	candidates := make([]string, 0)
	for _, v := range vm.Versions {
		if !strings.Contains(v, mcVer) {
			continue
		}
		if matchesChannel(v, channel) {
			candidates = append(candidates, v)
		}
	}

	// If no candidates found for the specific channel, try all matching versions
	if len(candidates) == 0 {
		for _, v := range vm.Versions {
			if strings.Contains(v, mcVer) {
				candidates = append(candidates, v)
			}
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no npm version found matching Minecraft version %s for channel %s", minecraftVersion, channel)
	}

	// Sort candidates: highest version first
	sort.Slice(candidates, func(i, j int) bool {
		return compareSemver(candidates[i], candidates[j]) > 0
	})

	return candidates[0], nil
}

// matchesChannel checks if a version string matches the given channel
func matchesChannel(version string, channel string) bool {
	switch channel {
	case "stable":
		// Stable: no beta, no preview, no rc
		return !strings.Contains(version, "-beta") &&
			!strings.Contains(version, "-preview") &&
			!strings.Contains(version, "-rc")
	case "beta":
		// Beta: has -beta but NOT -preview
		return strings.Contains(version, "-beta") && !strings.Contains(version, "-preview")
	case "preview":
		// Preview: has -preview (could be beta+preview or rc+preview)
		return strings.Contains(version, "-preview")
	default:
		return true
	}
}

// resolveLatestStable finds the highest stable version
func resolveLatestStable(vm *VersionMatrix) (string, error) {
	stableCandidates := make([]string, 0)
	for _, v := range vm.Versions {
		if !strings.Contains(v, "-beta") && !strings.Contains(v, "-preview") && !strings.Contains(v, "-rc") {
			stableCandidates = append(stableCandidates, v)
		}
	}
	if len(stableCandidates) == 0 {
		return "", fmt.Errorf("no stable version found for module %s", vm.Module)
	}
	sort.Slice(stableCandidates, func(i, j int) bool {
		return compareSemver(stableCandidates[i], stableCandidates[j]) > 0
	})
	return NormalizeVersion(stableCandidates[0]), nil
}

// resolveLatestBeta finds the highest beta version (has -beta but NOT -preview)
func resolveLatestBeta(vm *VersionMatrix) (string, error) {
	betaCandidates := make([]string, 0)
	for _, v := range vm.Versions {
		if strings.Contains(v, "-beta") && !strings.Contains(v, "-preview") {
			betaCandidates = append(betaCandidates, v)
		}
	}
	if len(betaCandidates) > 0 {
		sort.Slice(betaCandidates, func(i, j int) bool {
			return compareSemver(betaCandidates[i], betaCandidates[j]) > 0
		})
		return NormalizeVersion(betaCandidates[0]), nil
	}
	// Fallback to latest tag if no pure beta
	if latest, ok := vm.Tags["latest"]; ok {
		return NormalizeVersion(latest), nil
	}
	return "", fmt.Errorf("no beta version found for module %s", vm.Module)
}

// resolveLatestPreview finds the highest preview version (has -preview)
func resolveLatestPreview(vm *VersionMatrix) (string, error) {
	previewCandidates := make([]string, 0)
	for _, v := range vm.Versions {
		if strings.Contains(v, "-preview") {
			previewCandidates = append(previewCandidates, v)
		}
	}
	if len(previewCandidates) > 0 {
		sort.Slice(previewCandidates, func(i, j int) bool {
			return compareSemver(previewCandidates[i], previewCandidates[j]) > 0
		})
		return NormalizeVersion(previewCandidates[0]), nil
	}
	// Fallback to beta tag if no preview
	if beta, ok := vm.Tags["beta"]; ok {
		return NormalizeVersion(beta), nil
	}
	return "", fmt.Errorf("no preview version found for module %s", vm.Module)
}

// compareSemver compares two version strings. Returns 1 if a > b, -1 if a < b, 0 if equal.
func compareSemver(a, b string) int {
	// Strip any pre-release suffixes for base comparison
	baseA := strings.Split(a, "-")[0]
	baseB := strings.Split(b, "-")[0]

	partsA := strings.Split(baseA, ".")
	partsB := strings.Split(baseB, ".")

	for i := 0; i < len(partsA) && i < len(partsB); i++ {
		// Simple integer comparison of version parts
		var numA, numB int
		fmt.Sscanf(partsA[i], "%d", &numA)
		fmt.Sscanf(partsB[i], "%d", &numB)
		if numA > numB {
			return 1
		}
		if numA < numB {
			return -1
		}
	}

	// If base versions are equal, prefer versions with more parts (more specific)
	if len(partsA) != len(partsB) {
		return len(partsA) - len(partsB)
	}

	// If still equal, prefer stable over pre-release
	aHasPre := strings.Contains(a, "-")
	bHasPre := strings.Contains(b, "-")
	if !aHasPre && bHasPre {
		return 1
	}
	if aHasPre && !bHasPre {
		return -1
	}

	// Lexicographic comparison of full version string as fallback
	if a > b {
		return 1
	}
	if a < b {
		return -1
	}
	return 0
}

// ResolveVersion maps a Minecraft version string to the best matching npm version.
// Supports "latest" tag.
func ResolveVersion(vm *VersionMatrix, minecraftVersion string) (string, error) {
	if minecraftVersion == "latest" {
		if latest, ok := vm.Tags["latest"]; ok {
			return latest, nil
		}
		return "", fmt.Errorf("no latest tag found for module %s", vm.Module)
	}

	// Clean input: remove leading "v" if present
	mcVer := strings.TrimPrefix(minecraftVersion, "v")

	// Filter versions that start with the minecraft version prefix
	// npm versions look like "1.21.60", "1.21.60-stable", "1.21.60-beta"
	candidates := make([]string, 0)
	for _, v := range vm.Versions {
		if strings.HasPrefix(v, mcVer) {
			candidates = append(candidates, v)
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no npm version found matching Minecraft version %s", minecraftVersion)
	}

	// Sort candidates: stable first, then by semver rules
	sort.Slice(candidates, func(i, j int) bool {
		vi, vj := candidates[i], candidates[j]
		// Prefer versions without "-" (stable) over prerelease
		iStable := !strings.Contains(vi, "-")
		jStable := !strings.Contains(vj, "-")
		if iStable != jStable {
			return iStable
		}
		// Use proper semver comparison
		return compareSemver(vi, vj) > 0
	})

	return candidates[0], nil
}

// ExtractTypes pulls a TypeScript definition block and its referenced types from .d.ts content.
// It finds the primary symbol, then recursively gathers types referenced in its body.
func ExtractTypes(dts []byte, query string) (string, error) {
	content := string(dts)
	lines := strings.Split(content, "\n")

	// Find the primary definition
	start, end, ok := findDefinitionBlock(lines, query)
	if !ok {
		return "", fmt.Errorf("symbol %q not found in type definitions", query)
	}

	// Extract the block
	blockLines := lines[start:end]
	result := strings.Join(blockLines, "\n")

	// Gather referenced types from the block
	refs := extractReferencedTypes(strings.Join(blockLines, "\n"))

	// For each referenced type not yet included, find and append it
	included := map[string]bool{query: true}
	for _, ref := range refs {
		if included[ref] {
			continue
		}
		rs, re, ok := findDefinitionBlock(lines, ref)
		if !ok {
			continue
		}
		result += "\n\n" + strings.Join(lines[rs:re], "\n")
		included[ref] = true
	}

	return result, nil
}

// findDefinitionBlock locates a class/interface/enum/type/function by name.
// Returns line indices [start, end) of the block.
func findDefinitionBlock(lines []string, name string) (int, int, bool) {
	// Pattern: export (declare)? (class|interface|enum|type|function|namespace|module|const|let|var) Name ...
	// Also support: Name: or Name = in object types, but we aim for top-level.
	re := regexp.MustCompile(`^\s*(export\s+)?(declare\s+)?(abstract\s+)?(class|interface|enum|type|function|namespace|module|const|let|var)\s+` + regexp.QuoteMeta(name) + `[\s{:(=<;]`)

	start := -1
	for i, line := range lines {
		if re.MatchString(line) {
			start = i
			break
		}
	}
	if start == -1 {
		return 0, 0, false
	}

	// Heuristic: if line ends with semicolon, it's a one-liner
	if strings.HasSuffix(strings.TrimSpace(lines[start]), ";") {
		return start, start + 1, true
	}

	// Count braces to find end of block
	braceCount := 0
	inBlock := false
	end := start + 1
	for i := start; i < len(lines); i++ {
		line := lines[i]
		for _, ch := range line {
			switch ch {
			case '{':
				braceCount++
				inBlock = true
			case '}':
				braceCount--
				if braceCount == 0 && inBlock {
					end = i + 1
					return start, end, true
				}
			}
		}
		// If no braces and line ends with semicolon (type alias, const, etc.)
		if !inBlock && strings.HasSuffix(strings.TrimSpace(line), ";") && i > start {
			end = i + 1
			return start, end, true
		}
	}

	return start, len(lines), true
}

// extractReferencedTypes finds capitalized identifiers that are likely types.
func extractReferencedTypes(body string) []string {
	// Match standalone capitalized identifiers (exclude keywords)
	re := regexp.MustCompile(`\b([A-Z][a-zA-Z0-9_]*)\b`)
	matches := re.FindAllStringSubmatch(body, -1)

	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, m := range matches {
		t := m[1]
		if seen[t] {
			continue
		}
		if isKeyword(t) {
			continue
		}
		seen[t] = true
		result = append(result, t)
	}
	return result
}

func isKeyword(s string) bool {
	switch s {
	case "const", "let", "var", "type", "interface", "class", "enum",
		"function", "namespace", "module", "export", "import", "from",
		"return", "if", "else", "for", "while", "switch", "case",
		"break", "continue", "new", "this", "super", "extends",
		"implements", "public", "private", "protected", "readonly",
		"static", "abstract", "async", "await", "yield", "throw",
		"try", "catch", "finally", "void", "null", "undefined",
		"true", "false", "number", "string", "boolean", "symbol",
		"any", "unknown", "never", "object", "Array", "Map", "Set",
		"Promise", "Date", "RegExp", "Error", "String", "Number",
		"Boolean", "Object", "Function", "Record", "Partial",
		"Required", "Readonly", "Pick", "Omit", "Exclude", "Extract",
		"NonNullable", "Parameters", "ReturnType", "InstanceType",
		"ThisParameterType", "OmitThisParameter", "ThisType",
		"Uppercase", "Lowercase", "Capitalize", "Uncapitalize",
		"ConstructorParameters", "Awaited", "PropertyKey", "IterableIterator",
		"Iterator", "Iterable", "AsyncIterable", "AsyncIterableIterator",
		"AsyncIterator", "ArrayLike", "ReadonlyArray", "ReadonlyMap",
		"ReadonlySet", "Intl", "JSON", "Math", "console", "Buffer",
		"NodeJS", "globalThis", "console.warn", "console.log", "console.error":
		return true
	}
	return false
}

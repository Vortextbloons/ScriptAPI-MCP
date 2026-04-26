package npm

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

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
		// Simple string compare works for same-prefix semver
		return vi > vj
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

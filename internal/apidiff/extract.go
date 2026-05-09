package apidiff

import (
	"regexp"
	"strings"
)

// BuildSymbolTable parses TypeScript .d.ts content into a SymbolTable.
// It extracts top-level exported declarations, their members, and builds
// fully-qualified names by expanding known root objects.
func BuildSymbolTable(dts []byte, module, version string) (*SymbolTable, error) {
	content := string(dts)
	lines := strings.Split(content, "\n")

	var roots []ExportedSymbol
	flat := make(map[string]ExportedSymbol)

	// Pass 1: Find all top-level exported declarations
	topLevel := findTopLevelDeclarations(lines)

	for _, decl := range topLevel {
		symbol := parseDeclaration(lines, decl.start, decl.end, "")
		if symbol != nil {
			roots = append(roots, *symbol)
			flattenSymbol(*symbol, "", &flat)
		}
	}

	// Pass 2: Expand dotted paths for known root objects (world, system, etc.)
	expandRootPaths(&roots, &flat)

	table := &SymbolTable{
		Module:  module,
		Version: version,
		Roots:   roots,
		Flat:    flat,
	}
	return table, nil
}

type declRange struct {
	start int
	end   int
	name  string
	kind  SymbolKind
}

func findTopLevelDeclarations(lines []string) []declRange {
	var decls []declRange
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*export\s+declare\s+(abstract\s+)?(class|interface|enum|namespace|function|type|const|let|var)\s+([A-Za-z_]\w*)`),
		regexp.MustCompile(`^\s*declare\s+(abstract\s+)?(class|interface|enum|namespace|function|type|const|let|var)\s+([A-Za-z_]\w*)`),
		regexp.MustCompile(`^\s*export\s+(abstract\s+)?(class|interface|enum|namespace|function|type|const|let|var)\s+([A-Za-z_]\w*)`),
	}

	for i, line := range lines {
		for _, re := range patterns {
			matches := re.FindStringSubmatch(line)
			if matches == nil {
				continue
			}
			kindStr := ""
			nameStr := ""
			for _, m := range matches {
				switch m {
				case "class", "interface", "enum", "namespace", "function", "type", "const", "let", "var":
					kindStr = m
				}
			}
			nameStr = matches[len(matches)-1]
			if kindStr == "" || nameStr == "" {
				continue
			}

			end := findBlockEnd(lines, i)
			decls = append(decls, declRange{
				start: i,
				end:   end,
				name:  nameStr,
				kind:  toSymbolKind(kindStr),
			})
			break
		}
	}
	return decls
}

func toSymbolKind(tsKind string) SymbolKind {
	switch tsKind {
	case "class":
		return KindClass
	case "interface":
		return KindInterface
	case "enum":
		return KindEnum
	case "namespace":
		return KindNamespace
	case "function":
		return KindFunction
	case "type":
		return KindType
	case "const", "let", "var":
		return KindVariable
	default:
		return KindVariable
	}
}

func findBlockEnd(lines []string, start int) int {
	line := strings.TrimSpace(lines[start])
	if strings.HasSuffix(line, ";") {
		return start + 1
	}

	braceCount := 0
	inBlock := false
	for i := start; i < len(lines); i++ {
		ln := lines[i]
		for _, ch := range ln {
			if ch == '{' {
				braceCount++
				inBlock = true
			} else if ch == '}' {
				braceCount--
				if braceCount == 0 && inBlock {
					return i + 1
				}
			}
		}
	}
	return len(lines)
}

func parseDeclaration(lines []string, start, end int, parent string) *ExportedSymbol {
	firstLine := strings.TrimSpace(lines[start])
	name := extractName(firstLine)
	if name == "" {
		return nil
	}

	kind := detectKind(firstLine)
	qualified := name
	if parent != "" {
		qualified = parent + "." + name
	}

	sig := extractSignature(lines, start, end)

	deprecated := false
	for i := start - 1; i >= 0 && i >= start-3; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if strings.Contains(trimmed, "@deprecated") {
			deprecated = true
			break
		}
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		break
	}

	symbol := &ExportedSymbol{
		Name:       qualified,
		Kind:       kind,
		Signature:  sig,
		Deprecated: deprecated,
		Parent:     parent,
	}

	if kind == KindClass || kind == KindInterface || kind == KindNamespace || kind == KindEnum {
		members := extractMembers(lines, start, end, qualified)
		symbol.Members = members
	}

	return symbol
}

func extractName(line string) string {
	re := regexp.MustCompile(`(?:export\s+)?(?:declare\s+)?(?:abstract\s+)?(?:class|interface|enum|namespace|function|type|const|let|var)\s+([A-Za-z_]\w*)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func detectKind(line string) SymbolKind {
	re := regexp.MustCompile(`(?:export\s+)?(?:declare\s+)?(?:abstract\s+)?(class|interface|enum|namespace|function|type|const|let|var)\b`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 2 {
		return toSymbolKind(matches[1])
	}
	return KindVariable
}

func extractSignature(lines []string, start, end int) string {
	var sb strings.Builder
	for i := start; i < end && i < start+5; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(trimmed)
	}
	sig := sb.String()
	if len(sig) > 200 {
		sig = sig[:200] + "..."
	}
	return sig
}

func extractMembers(lines []string, start, end int, qualified string) []ExportedSymbol {
	var members []ExportedSymbol
	memberRe := regexp.MustCompile(`^\s+(readonly\s+)?([A-Za-z_]\w*)\??\s*[:(=]`)
	deprecatedRe := regexp.MustCompile(`@deprecated`)

	for i := start + 1; i < end-1; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "/*") {
			continue
		}
		if line == "}" || line == "{" {
			continue
		}

		matches := memberRe.FindStringSubmatch(lines[i])
		if matches == nil {
			continue
		}
		memberName := matches[2]
		if memberName == "" {
			continue
		}

		deprecated := false
		for j := i - 1; j >= 0 && j >= i-2; j-- {
			commentLine := strings.TrimSpace(lines[j])
			if deprecatedRe.MatchString(commentLine) {
				deprecated = true
				break
			}
			if !strings.HasPrefix(commentLine, "//") && !strings.HasPrefix(commentLine, "*") {
				break
			}
		}

		member := ExportedSymbol{
			Name:       qualified + "." + memberName,
			Kind:       KindProperty,
			Deprecated: deprecated,
			Parent:     qualified,
		}

		if strings.Contains(lines[i], "(") && strings.Contains(lines[i], ")") {
			member.Kind = KindMethod
		}

		members = append(members, member)
	}
	return members
}

func flattenSymbol(symbol ExportedSymbol, parent string, flat *map[string]ExportedSymbol) {
	m := *flat
	key := symbol.Name
	if _, exists := m[key]; !exists {
		m[key] = symbol
	}
	for _, member := range symbol.Members {
		flattenSymbol(member, symbol.Name, flat)
	}
}

func expandRootPaths(roots *[]ExportedSymbol, flat *map[string]ExportedSymbol) {
	knownRoots := map[string]bool{
		"world":         true,
		"system":        true,
		"Player":        true,
		"Entity":        true,
		"Block":         true,
		"ItemStack":     true,
		"Dimension":     true,
		"Container":     true,
		"Scoreboard":    true,
		"CommandResult": true,
	}

	m := *flat
	for _, root := range *roots {
		m[root.Name] = root
		for _, member := range root.Members {
			m[member.Name] = member
			if knownRoots[root.Name] || root.Kind == KindNamespace {
				for _, sub := range member.Members {
					m[sub.Name] = sub
				}
			}
		}
	}
}

// GetFlatNames returns all fully-qualified symbol names from the table.
func (st *SymbolTable) GetFlatNames() []string {
	names := make([]string, 0, len(st.Flat))
	for name := range st.Flat {
		names = append(names, name)
	}
	return names
}

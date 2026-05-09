package snippets

import (
	"fmt"
	"strings"
)

// SnippetOutput is the result of generating a snippet.
type SnippetOutput struct {
	Files           map[string]string `json:"files"`
	RequiredModules []string          `json:"required_modules"`
	Notes           []string          `json:"notes"`
	Imports         []string          `json:"imports"`
}

// GenerateSnippet generates a code snippet for the given type and language.
// name is used as the {{name}} placeholder replacement (e.g. custom component name).
// moduleVersion, if non-empty, is prepended as a version comment.
// includeComments adds JSDoc/TSDoc above the handler.
func GenerateSnippet(snippetType, language, name, moduleVersion string, includeComments bool) (*SnippetOutput, error) {
	definition, ok := GetDefinition(snippetType)
	if !ok {
		return nil, fmt.Errorf("unknown snippet type: %s", snippetType)
	}

	// Validate language
	if language != "javascript" && language != "typescript" {
		language = "javascript"
	}

	// Select source and imports based on language
	var source string
	var imports []string
	var allImports []string
	fileExt := "js"

	if language == "typescript" {
		source = definition.TypeScript
		fileExt = "ts"
		imports = definition.TSImports
		allImports = append(definition.TSImports, definition.TSTypeImports...)
	} else {
		source = definition.JavaScript
		imports = definition.JSImports
		allImports = definition.JSImports
	}

	// Replace {{name}} placeholder
	if name != "" {
		source = strings.ReplaceAll(source, "{{name}}", name)
	} else if strings.Contains(source, "{{name}}") {
		source = strings.ReplaceAll(source, "{{name}}", "your_namespace:your_component")
	}

	// Build import lines
	var importSection string
	if len(imports) > 0 {
		importSection = fmt.Sprintf("import { %s } from \"@minecraft/server\";\n", strings.Join(imports, ", "))
	}
	if language == "typescript" && len(definition.TSTypeImports) > 0 {
		importSection += fmt.Sprintf("import type { %s } from \"@minecraft/server\";\n", strings.Join(definition.TSTypeImports, ", "))
	}

	// Build full source
	var fullSource strings.Builder

	// Version comment
	if moduleVersion != "" {
		fullSource.WriteString(fmt.Sprintf("// Target: @minecraft/server@%s\n", moduleVersion))
		fullSource.WriteString("\n")
	}

	// Imports
	fullSource.WriteString(importSection)
	fullSource.WriteString("\n")

	// Code body
	fullSource.WriteString(source)

	// File path
	filePath := fmt.Sprintf("src/main.%s", fileExt)

	files := map[string]string{
		filePath: fullSource.String(),
	}

	notes := make([]string, 0)

	return &SnippetOutput{
		Files:           files,
		RequiredModules: definition.RequiredModules,
		Notes:           notes,
		Imports:         allImports,
	}, nil
}

package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"
)

type ValidateScriptAPIUsageInput struct {
	ProjectPath string `json:"project_path" mcp:"required,description='Addon workspace containing source files, behavior pack manifest, and optional node_modules'"`
}
type APIUsageFinding struct {
	Severity string `json:"severity"`
	Rule     string `json:"rule"`
	File     string `json:"file,omitempty"`
	Module   string `json:"module,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Message  string `json:"message"`
}
type ValidateScriptAPIUsageOutput struct {
	Valid        bool              `json:"valid"`
	Modules      map[string]string `json:"modules"`
	FilesScanned []string          `json:"files_scanned"`
	Findings     []APIUsageFinding `json:"findings"`
	CacheNote    string            `json:"cache_note"`
}

var importRE = regexp.MustCompile(`(?m)(?:import(?:\s+type)?\s+(?:([^;]+?)\s+from\s+)?|require\()['\"](@minecraft/[a-zA-Z0-9-]+)['\"]`)
var namesRE = regexp.MustCompile(`[A-Za-z_$][A-Za-z0-9_$]*`)

func RegisterValidateScriptAPIUsage(server *mcp.Server) error {
	return server.RegisterTool("validate_script_api_usage", "Scans JavaScript/TypeScript sources against the behavior-pack manifest and installed @minecraft declarations. Detects undeclared imports, unavailable symbols, deprecated APIs, and output/entrypoint mismatches. It is read-only and works offline from node_modules.", func(a ValidateScriptAPIUsageInput) (*mcp.ToolResponse, error) { return handleValidateScriptAPIUsage(a) })
}
func handleValidateScriptAPIUsage(args ValidateScriptAPIUsageInput) (*mcp.ToolResponse, error) {
	inspect, err := inspectAddonWorkspace(args.ProjectPath)
	if err != nil {
		return toolErrorResponse("VALIDATION_FAILED", err.Error(), false), nil
	}
	declared := map[string]bool{}
	versions := map[string]string{}
	for _, m := range inspect.Modules {
		declared[m] = true
		if installed, e := getModuleVersion(args.ProjectPath, m); e == nil {
			versions[m] = installed
		}
	}
	out := ValidateScriptAPIUsageOutput{Valid: true, Modules: versions, Findings: []APIUsageFinding{}, CacheNote: "Validation uses local node_modules declarations; no network request is required."}
	_ = filepath.Walk(filepath.Join(args.ProjectPath, "src"), func(path string, info os.FileInfo, e error) error {
		if e != nil || info == nil || info.IsDir() || (!strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".js")) {
			return nil
		}
		out.FilesScanned = append(out.FilesScanned, filepath.ToSlash(strings.TrimPrefix(path, args.ProjectPath+string(filepath.Separator))))
		data, _ := os.ReadFile(path)
		for _, match := range importRE.FindAllStringSubmatch(string(data), -1) {
			names, module := match[1], match[2]
			if !declared[module] {
				out.Findings = append(out.Findings, APIUsageFinding{Severity: "error", Rule: "undeclared_module_import", File: out.FilesScanned[len(out.FilesScanned)-1], Module: module, Message: "import is not declared in behavior pack manifest dependencies"})
				out.Valid = false
			}
			dts, readErr := os.ReadFile(filepath.Join(args.ProjectPath, "node_modules", filepath.FromSlash(module), "index.d.ts"))
			if readErr != nil {
				out.Findings = append(out.Findings, APIUsageFinding{Severity: "warning", Rule: "missing_type_definitions", File: out.FilesScanned[len(out.FilesScanned)-1], Module: module, Message: "local type definitions are unavailable; symbol validation skipped"})
				continue
			}
			for _, symbol := range namesRE.FindAllString(names, -1) {
				if symbol == "as" || symbol == "type" || symbol == "default" {
					continue
				}
				if !regexp.MustCompile(`(?m)\b` + regexp.QuoteMeta(symbol) + `\b`).Match(dts) {
					out.Findings = append(out.Findings, APIUsageFinding{Severity: "warning", Rule: "unknown_api_symbol", File: out.FilesScanned[len(out.FilesScanned)-1], Module: module, Symbol: symbol, Message: "symbol was not found in installed declaration file"})
				} else if regexp.MustCompile(`(?s)@deprecated.{0,300}\b` + regexp.QuoteMeta(symbol) + `\b`).Match(dts) {
					out.Findings = append(out.Findings, APIUsageFinding{Severity: "warning", Rule: "deprecated_api_symbol", File: out.FilesScanned[len(out.FilesScanned)-1], Module: module, Symbol: symbol, Message: "symbol is marked deprecated in installed declarations"})
				}
			}
		}
		return nil
	})
	if inspect.Entrypoint == "" || !fileExists(filepath.Join(args.ProjectPath, filepath.FromSlash(inspect.BPSource), filepath.FromSlash(inspect.Entrypoint))) {
		out.Valid = false
		out.Findings = append(out.Findings, APIUsageFinding{Severity: "error", Rule: "script_output_mismatch", Message: "manifest script entrypoint is missing from the behavior pack"})
	}
	sort.Strings(out.FilesScanned)
	b, _ := json.MarshalIndent(out, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
}
func getModuleVersion(project, module string) (string, error) {
	b, e := os.ReadFile(filepath.Join(project, "node_modules", filepath.FromSlash(module), "package.json"))
	if e != nil {
		return "", e
	}
	var p struct {
		Version string `json:"version"`
	}
	e = json.Unmarshal(b, &p)
	return p.Version, e
}

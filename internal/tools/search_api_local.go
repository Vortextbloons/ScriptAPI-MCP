package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

const (
	minecraftScope         = "@minecraft/"
	localLargeFileLines    = 50000
	localMaxTotalMatches   = 10000
	localMaxTypeTextBytes  = 16384
)

type localSearchMatch struct {
	Module   string `json:"module"`
	Version  string `json:"version"`
	Line     int    `json:"line"`
	LineText string `json:"line_text"`
	Kind     string `json:"kind,omitempty"`
	Name     string `json:"name,omitempty"`
	Truncated bool  `json:"truncated,omitempty"`
}

type skippedModule struct {
	Module string `json:"module"`
	Reason string `json:"reason"`
}

type localSearchResult struct {
	matches      []localSearchMatch
	total        int
	truncated    bool
	matchCapHit  bool
	largeFiles   []string
	skipped      []skippedModule
}

func handleSearchAPILocal(args SearchAPIInput) (*mcp.ToolResponse, error) {
	projectPath := strings.TrimSpace(args.ProjectPath)
	if projectPath == "" {
		return toolErrorResponse("INVALID_INPUT", "project_path is required when source=local", false, "Pass the absolute path to the project root that contains node_modules"), nil
	}
	cleaned := filepath.Clean(projectPath)
	info, err := os.Stat(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			return toolErrorResponse("PROJECT_PATH_NOT_FOUND", fmt.Sprintf("project_path %q does not exist", projectPath), false, "Verify the path to your Bedrock addon project"), nil
		}
		return toolErrorResponse("PROJECT_PATH_NOT_FOUND", fmt.Sprintf("cannot access project_path %q: %v", projectPath, err), false), nil
	}
	if !info.IsDir() {
		return toolErrorResponse("PROJECT_PATH_NOT_FOUND", fmt.Sprintf("project_path %q is not a directory", projectPath), false), nil
	}

	modules, errResp, err := resolveLocalModules(cleaned, args.Modules)
	if err != nil {
		return nil, err
	}
	if errResp != nil {
		return errResp, nil
	}

	mode := strings.ToLower(strings.TrimSpace(args.Mode))
	if mode == "" {
		mode = "types"
	}
	if mode != "types" && mode != "members" && mode != "index" {
		return toolErrorResponse("INVALID_INPUT", fmt.Sprintf("unknown mode %q (use types, members, or index)", args.Mode), false), nil
	}

	limit, lerr := clampLimit(args.Limit)
	if lerr != nil {
		return toolErrorResponse("INVALID_INPUT", lerr.Error(), false), nil
	}
	offset := args.Offset
	if offset < 0 {
		offset = 0
	}

	query := strings.TrimSpace(args.Query)
	if query == "" {
		return toolErrorResponse("INVALID_INPUT", "query is required", false), nil
	}

	var result *localSearchResult
	switch mode {
	case "members":
		result = searchLocalMembers(modules, query, limit, offset)
	case "index":
		result = searchLocalIndex(modules, query, limit, offset)
	default:
		result = searchLocalTypes(modules, query)
	}

	if result.total == 0 {
		if len(result.skipped) == len(modules) {
			b, _ := json.MarshalIndent(map[string]any{
				"source":           "local",
				"project_path":     projectPath,
				"query":            query,
				"mode":             mode,
				"modules_searched": []string{},
				"skipped_modules":  result.skipped,
			}, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		}
		return toolErrorResponse("NO_MATCH", fmt.Sprintf("no results matched query %q in installed @minecraft/* modules", args.Query), false,
			"Try mode=members for broader grep-style search",
			"Try mode=index to browse exported symbols"), nil
	}

	modulesSearched := moduleNames(modules)
	searched := make([]string, 0, len(modulesSearched))
	for _, name := range modulesSearched {
		skipped := false
		for _, s := range result.skipped {
			if s.Module == name {
				skipped = true
				break
			}
		}
		if !skipped {
			searched = append(searched, name)
		}
	}

	envelope := map[string]any{
		"source":           "local",
		"project_path":     projectPath,
		"query":            query,
		"mode":             mode,
		"modules_searched": searched,
		"total_matches":    result.total,
		"offset":           offset,
		"limit":            limit,
		"truncated":        result.truncated,
	}
	if len(result.skipped) > 0 {
		envelope["skipped_modules"] = result.skipped
	}
	if len(result.largeFiles) > 0 {
		envelope["large_files"] = result.largeFiles
	}
	if result.matchCapHit {
		envelope["match_cap_hit"] = true
		envelope["match_cap"] = localMaxTotalMatches
	}

	switch mode {
	case "types":
		envelope["results"] = result.matches
	default:
		envelope["matches"] = result.matches
	}

	b, _ := json.MarshalIndent(envelope, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
}

func moduleNames(modules []npm.InstalledModule) []string {
	names := make([]string, 0, len(modules))
	for _, m := range modules {
		names = append(names, m.Name)
	}
	return names
}

func resolveLocalModules(projectPath string, explicit []string) ([]npm.InstalledModule, *mcp.ToolResponse, error) {
	if len(explicit) > 0 {
		out := make([]npm.InstalledModule, 0, len(explicit))
		for _, name := range explicit {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if !strings.HasPrefix(name, minecraftScope) {
				return nil, toolErrorResponse("INVALID_INPUT",
					fmt.Sprintf("module %q is not under the @minecraft/ scope; source=local only searches installed @minecraft/*", name),
					false,
					"Use a module name like @minecraft/server",
					"Omit 'modules' to auto-scan all installed @minecraft/*"), nil
			}
			mod, err := npm.GetInstalledModule(projectPath, name)
			if err != nil {
				return nil, toolErrorResponse("MODULE_NOT_FOUND",
					fmt.Sprintf("module %q is not installed under %s: %v", name, projectPath, err),
					false,
					"Run npm install in your project",
					"Verify the module name is correct"), nil
			}
			out = append(out, *mod)
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out, nil, nil
	}

	out, err := npm.GetInstalledMinecraftModules(projectPath)
	if err != nil {
		return nil, toolErrorResponse("NO_MODULES_INSTALLED",
			fmt.Sprintf("failed to read installed @minecraft/* modules: %v", err),
			false), nil
	}
	if len(out) == 0 {
		return nil, toolErrorResponse("NO_MODULES_INSTALLED",
			fmt.Sprintf("no @minecraft/* modules found under %s", filepath.Join(projectPath, "node_modules")),
			false,
			"Run npm install in your project",
			"Verify the project has @minecraft/* dependencies declared in package.json"), nil
	}
	return out, nil, nil
}

func findDTSFile(modulePath string) (string, error) {
	primary := filepath.Join(modulePath, "index.d.ts")
	if info, err := os.Stat(primary); err == nil && !info.IsDir() {
		return primary, nil
	}
	entries, err := os.ReadDir(modulePath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", modulePath, err)
	}
	var candidates []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".d.ts") {
			candidates = append(candidates, filepath.Join(modulePath, e.Name()))
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no .d.ts file found in %s", modulePath)
	}
	sort.Strings(candidates)
	return candidates[0], nil
}

type moduleDTS struct {
	lines      []string
	largeFile  bool
}

func readModuleDTS(m npm.InstalledModule) (moduleDTS, error) {
	path, err := findDTSFile(m.PackagePath)
	if err != nil {
		return moduleDTS{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return moduleDTS{}, fmt.Errorf("read %s: %w", path, err)
	}
	lines := strings.Split(string(content), "\n")
	return moduleDTS{
		lines:     lines,
		largeFile: len(lines) > localLargeFileLines,
	}, nil
}

func searchLocalMembers(modules []npm.InstalledModule, query string, limit, offset int) *localSearchResult {
	q := strings.ToLower(query)
	state := newLocalMatchCollector(limit, offset)
	for _, m := range modules {
		dts, err := readModuleDTS(m)
		if err != nil {
			state.skip(m.Name, err.Error())
			continue
		}
		if dts.largeFile {
			state.noteLargeFile(m.Name)
		}
		for i, line := range dts.lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if !strings.Contains(strings.ToLower(trimmed), q) {
				continue
			}
			if !state.accept(localSearchMatch{
				Module:   m.Name,
				Version:  m.Version,
				Line:     i + 1,
				LineText: trimmed,
			}) {
				return state.result()
			}
		}
	}
	return state.result()
}

var localExportRegex = regexp.MustCompile(`^\s*export\s+(?:declare\s+)?(?:abstract\s+)?(class|interface|enum|type|function|namespace|module|const|let|var)\s+([A-Za-z_][A-Za-z0-9_]*)`)

func searchLocalIndex(modules []npm.InstalledModule, query string, limit, offset int) *localSearchResult {
	q := strings.ToLower(query)
	state := newLocalMatchCollector(limit, offset)
	for _, m := range modules {
		dts, err := readModuleDTS(m)
		if err != nil {
			state.skip(m.Name, err.Error())
			continue
		}
		if dts.largeFile {
			state.noteLargeFile(m.Name)
		}
		for i, line := range dts.lines {
			match := localExportRegex.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			kind, name := match[1], match[2]
			if !strings.Contains(strings.ToLower(name), q) {
				continue
			}
			if !state.accept(localSearchMatch{
				Module:   m.Name,
				Version:  m.Version,
				Line:     i + 1,
				LineText: strings.TrimSpace(line),
				Kind:     kind,
				Name:     name,
			}) {
				return state.result()
			}
		}
	}
	return state.result()
}

func searchLocalTypes(modules []npm.InstalledModule, query string) *localSearchResult {
	result := &localSearchResult{matches: make([]localSearchMatch, 0)}
	for _, m := range modules {
		dts, err := readModuleDTS(m)
		if err != nil {
			result.skipped = append(result.skipped, skippedModule{Module: m.Name, Reason: err.Error()})
			continue
		}
		if dts.largeFile {
			result.largeFiles = append(result.largeFiles, m.Name)
		}
		body, err := npm.ExtractTypes([]byte(strings.Join(dts.lines, "\n")), query)
		if err != nil {
			continue
		}
		trimmedBody := strings.TrimSpace(body)
		if trimmedBody == "" {
			continue
		}
		match := localSearchMatch{
			Module:   m.Name,
			Version:  m.Version,
			Line:     0,
			LineText: trimmedBody,
		}
		if len(trimmedBody) > localMaxTypeTextBytes {
			match.LineText = trimmedBody[:localMaxTypeTextBytes] + "\n... (truncated)"
			match.Truncated = true
		}
		result.matches = append(result.matches, match)
	}
	result.total = len(result.matches)
	return result
}

type localMatchCollector struct {
	limit       int
	offset      int
	page        []localSearchMatch
	total       int
	truncated   bool
	matchCapHit bool
	largeFiles  []string
	skipped     []skippedModule
}

func newLocalMatchCollector(limit, offset int) *localMatchCollector {
	return &localMatchCollector{
		limit:  limit,
		offset: offset,
		page:   make([]localSearchMatch, 0, limit),
	}
}

func (c *localMatchCollector) skip(module, reason string) {
	c.skipped = append(c.skipped, skippedModule{Module: module, Reason: reason})
}

func (c *localMatchCollector) noteLargeFile(module string) {
	for _, name := range c.largeFiles {
		if name == module {
			return
		}
	}
	c.largeFiles = append(c.largeFiles, module)
}

// accept records a match. Returns false when scanning can stop early.
func (c *localMatchCollector) accept(match localSearchMatch) bool {
	matchIdx := c.total
	if matchIdx >= localMaxTotalMatches {
		c.matchCapHit = true
		return false
	}

	if matchIdx >= c.offset && len(c.page) < c.limit {
		c.page = append(c.page, match)
	}
	c.total++
	return matchIdx+1 < localMaxTotalMatches
}

func (c *localMatchCollector) result() *localSearchResult {
	truncated := c.truncated || (len(c.page) >= c.limit && c.total > c.offset+len(c.page))
	return &localSearchResult{
		matches:     c.page,
		total:       c.total,
		truncated:   truncated,
		matchCapHit: c.matchCapHit,
		largeFiles:  c.largeFiles,
		skipped:     c.skipped,
	}
}

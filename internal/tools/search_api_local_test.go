package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcp "github.com/metoro-io/mcp-golang"
)

// writeModule creates a fake installed @minecraft/* module with a package.json
// and an index.d.ts containing the provided TypeScript source.
func writeModule(t *testing.T, projectPath, name, version, dts string) {
	t.Helper()
	dir := filepath.Join(projectPath, "node_modules", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	pkg := []byte(`{"name":"` + name + `","version":"` + version + `"}`)
	if err := os.WriteFile(filepath.Join(dir, "package.json"), pkg, 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.d.ts"), []byte(dts), 0o644); err != nil {
		t.Fatalf("write index.d.ts: %v", err)
	}
}

func searchAPIText(t *testing.T, resp *mcp.ToolResponse) string {
	t.Helper()
	if resp == nil || len(resp.Content) == 0 || resp.Content[0].TextContent == nil {
		t.Fatalf("response has no text content: %#v", resp)
	}
	return resp.Content[0].TextContent.Text
}

func runLocal(t *testing.T, args SearchAPIInput) map[string]any {
	t.Helper()
	resp, err := handleSearchAPILocal(args)
	if err != nil {
		t.Fatalf("handleSearchAPILocal: %v", err)
	}
	body := searchAPIText(t, resp)
	var env map[string]any
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("unmarshal body: %v\nbody=%s", err, body)
	}
	if errEnv, ok := env["error"]; ok {
		t.Fatalf("expected success, got error envelope: %#v", errEnv)
	}
	return env
}

func runLocalError(t *testing.T, args SearchAPIInput) map[string]any {
	t.Helper()
	resp, err := handleSearchAPILocal(args)
	if err != nil {
		t.Fatalf("handleSearchAPILocal returned Go error (should be tool envelope): %v", err)
	}
	body := searchAPIText(t, resp)
	var env map[string]any
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("unmarshal body: %v\nbody=%s", err, body)
	}
	if env["ok"] != false {
		t.Fatalf("expected ok=false error envelope, got: %v", env)
	}
	return env
}

func TestSearchLocalMembers_SingleModule(t *testing.T) {
	tmp := t.TempDir()
	dts := `/**
 * @minecraft/server module.
 */
declare module "@minecraft/server" {
    export class Player {
        name: string;
    }
    export interface AfterEvents {
        playerBreakBlock: PlayerBreakBlockEvent;
    }
    export class PlayerBreakBlockEvent {
        player: Player;
    }
}
`
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta.1.26.14-stable", dts)

	env := runLocal(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "playerBreakBlock",
		Mode:        "members",
	})

	if env["source"] != "local" {
		t.Fatalf("source = %v", env["source"])
	}
	if env["mode"] != "members" {
		t.Fatalf("mode = %v", env["mode"])
	}
	if env["total_matches"].(float64) != 2 {
		t.Fatalf("total_matches = %v, want 2 (interface field + class declaration)", env["total_matches"])
	}
	if env["offset"].(float64) != 0 || env["limit"].(float64) != 50 {
		t.Fatalf("offset/limit defaults wrong: %v/%v", env["offset"], env["limit"])
	}
	mods := env["modules_searched"].([]any)
	if len(mods) != 1 || mods[0] != "@minecraft/server" {
		t.Fatalf("modules_searched = %v", mods)
	}
	matches := env["matches"].([]any)
	if len(matches) == 0 {
		t.Fatal("expected at least one match")
	}
	first := matches[0].(map[string]any)
	if first["module"] != "@minecraft/server" {
		t.Fatalf("match.module = %v", first["module"])
	}
	if first["version"] != "2.7.0-beta.1.26.14-stable" {
		t.Fatalf("match.version = %v", first["version"])
	}
	if !strings.Contains(strings.ToLower(first["line_text"].(string)), "playerbreakblock") {
		t.Fatalf("line_text = %q", first["line_text"])
	}
	if first["line"].(float64) < 1 {
		t.Fatalf("line = %v", first["line"])
	}
}

func TestSearchLocalMembers_CrossModule(t *testing.T) {
	tmp := t.TempDir()
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta", `export class Player {}
export interface AfterEvents { playerBreakBlock: any; }
`)
	writeModule(t, tmp, "@minecraft/server-ui", "1.12.0-beta", `export class Player {} export interface ActionForm { playerBreakBlock?: boolean; }
`)

	env := runLocal(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Player",
		Mode:        "members",
	})

	mods := env["modules_searched"].([]any)
	if len(mods) != 2 {
		t.Fatalf("modules_searched = %v", mods)
	}
	if env["total_matches"].(float64) < 3 {
		t.Fatalf("total_matches = %v, want >=3", env["total_matches"])
	}
}

func TestSearchLocalIndex_ExportsOnly(t *testing.T) {
	tmp := t.TempDir()
	dts := `declare module "@minecraft/server" {
    export class Player { name: string; }
    export interface IPlayerRef { id: string; }
    export enum GameMode { Survival, Creative }
    export type Entity = Player | string;
    export function createPlayer(): Player;
    const internal: number = 0;
    interface NotExported {}
    export namespace World {
        export class Chunk {}
    }
}
`
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta", dts)

	env := runLocal(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Player",
		Mode:        "index",
	})

	entries := env["matches"].([]any)
	if len(entries) == 0 {
		t.Fatal("expected matches for Player query")
	}
	kinds := make(map[string]bool)
	names := make(map[string]bool)
	for _, e := range entries {
		m := e.(map[string]any)
		kinds[m["kind"].(string)] = true
		names[m["name"].(string)] = true
	}
	if !kinds["class"] {
		t.Fatalf("expected class kind, got %v", kinds)
	}
	if !names["Player"] {
		t.Fatalf("expected Player name, got %v", names)
	}
	for n := range names {
		if n == "NotExported" {
			t.Fatalf("non-exported name leaked into results: %v", names)
		}
	}
}

func TestSearchLocalTypes_ReusesExtractTypes(t *testing.T) {
	tmp := t.TempDir()
	dts := `export class Player {
    name: string;
    sendMessage(msg: string): void;
}
export interface PlayerBreakBlockEvent {
    player: Player;
}
`
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta", dts)

	env := runLocal(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Player",
		Mode:        "types",
	})

	results := env["results"].([]any)
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	body := results[0].(map[string]any)["line_text"].(string)
	if !strings.Contains(body, "class Player") {
		t.Fatalf("result body missing class Player: %q", body)
	}
}

func TestSearchLocal_AutoScan(t *testing.T) {
	tmp := t.TempDir()
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta", `export class Player {}
`)
	writeModule(t, tmp, "@minecraft/server-ui", "1.12.0-beta", `export class Form {}
`)
	// Drop a non-minecraft package to ensure it is not auto-scanned.
	bogus := filepath.Join(tmp, "node_modules", "lodash")
	if err := os.MkdirAll(bogus, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bogus, "package.json"), []byte(`{"name":"lodash","version":"4.17.21"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	env := runLocal(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Form",
		Mode:        "index",
	})

	mods := env["modules_searched"].([]any)
	if len(mods) != 2 {
		t.Fatalf("modules_searched = %v, want 2 (@minecraft/* only)", mods)
	}
}

func TestSearchLocal_PathSafety_RejectsNonMinecraftModule(t *testing.T) {
	tmp := t.TempDir()
	env := runLocalError(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Player",
		Mode:        "members",
		Modules:     []string{"lodash"},
	})
	if env["error"].(map[string]any)["code"] != "INVALID_INPUT" {
		t.Fatalf("expected INVALID_INPUT, got %v", env["error"])
	}
}

func TestSearchLocal_PathSafety_RejectsMissingProject(t *testing.T) {
	env := runLocalError(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: "C:/definitely/not/a/real/path/xyz123",
		Query:       "Player",
		Mode:        "members",
	})
	code := env["error"].(map[string]any)["code"]
	if code != "PROJECT_PATH_NOT_FOUND" {
		t.Fatalf("expected PROJECT_PATH_NOT_FOUND, got %v", code)
	}
}

func TestSearchLocal_EmptyNodeModules(t *testing.T) {
	tmp := t.TempDir()
	// Create node_modules/@minecraft but leave it empty.
	if err := os.MkdirAll(filepath.Join(tmp, "node_modules", "@minecraft"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	env := runLocalError(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Player",
		Mode:        "members",
	})
	if env["error"].(map[string]any)["code"] != "NO_MODULES_INSTALLED" {
		t.Fatalf("expected NO_MODULES_INSTALLED, got %v", env["error"])
	}
}

func TestSearchLocal_MissingNodeModules(t *testing.T) {
	tmp := t.TempDir()
	// No node_modules directory at all.
	env := runLocalError(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Player",
		Mode:        "members",
	})
	if env["error"].(map[string]any)["code"] != "NO_MODULES_INSTALLED" {
		t.Fatalf("expected NO_MODULES_INSTALLED, got %v", env["error"])
	}
}

func TestSearchLocal_RequiresProjectPath(t *testing.T) {
	env := runLocalError(t, SearchAPIInput{
		Source: "local",
		Query:  "Player",
		Mode:   "members",
	})
	if env["error"].(map[string]any)["code"] != "INVALID_INPUT" {
		t.Fatalf("expected INVALID_INPUT, got %v", env["error"])
	}
}

func TestSearchLocal_LimitCap(t *testing.T) {
	tmp := t.TempDir()
	var b strings.Builder
	for i := 0; i < 250; i++ {
		b.WriteString("export class Widget")
		b.WriteString(string(rune('A' + (i % 26))))
		b.WriteString(" {}\n")
	}
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta", b.String())

	// limit > max should error.
	env := runLocalError(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Widget",
		Mode:        "members",
		Limit:       999,
	})
	if env["error"].(map[string]any)["code"] != "INVALID_INPUT" {
		t.Fatalf("expected INVALID_INPUT for limit>200, got %v", env["error"])
	}

	// limit=0 -> default 50.
	env = runLocal(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Widget",
		Mode:        "members",
		Limit:       0,
	})
	if env["limit"].(float64) != 50 {
		t.Fatalf("default limit = %v, want 50", env["limit"])
	}
	matches := env["matches"].([]any)
	if len(matches) != 50 {
		t.Fatalf("len(matches) = %d, want 50 (default cap)", len(matches))
	}
	if env["truncated"] != true {
		t.Fatalf("truncated = %v, want true", env["truncated"])
	}

	// limit=10 -> exactly 10.
	env = runLocal(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Widget",
		Mode:        "members",
		Limit:       10,
	})
	if len(env["matches"].([]any)) != 10 {
		t.Fatalf("len(matches) = %d, want 10", len(env["matches"].([]any)))
	}
}

func TestSearchLocal_OffsetPagination(t *testing.T) {
	tmp := t.TempDir()
	var b strings.Builder
	for i := 0; i < 100; i++ {
		b.WriteString("export class Marker")
		b.WriteString(string(rune('A' + (i % 26))))
		b.WriteString(" {}\n")
	}
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta", b.String())

	page1 := runLocal(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Marker",
		Mode:        "members",
		Limit:       20,
		Offset:      0,
	})
	page2 := runLocal(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Marker",
		Mode:        "members",
		Limit:       20,
		Offset:      20,
	})
	if page1["total_matches"].(float64) != 100 {
		t.Fatalf("total_matches = %v, want 100", page1["total_matches"])
	}
	if page1["truncated"] != true {
		t.Fatalf("page1 truncated = %v", page1["truncated"])
	}
	p1 := page1["matches"].([]any)
	p2 := page2["matches"].([]any)
	if len(p1) != 20 || len(p2) != 20 {
		t.Fatalf("page sizes = %d/%d, want 20/20", len(p1), len(p2))
	}
	// Pages must be disjoint.
	l1 := p1[0].(map[string]any)["line"].(float64)
	l2 := p2[0].(map[string]any)["line"].(float64)
	if l2 <= l1 {
		t.Fatalf("page2 first line (%v) should be after page1 first line (%v)", l2, l1)
	}
}

func TestSearchLocal_MissingQuery(t *testing.T) {
	tmp := t.TempDir()
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta", `export class Player {}
`)
	env := runLocalError(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "",
		Mode:        "members",
	})
	if env["error"].(map[string]any)["code"] != "INVALID_INPUT" {
		t.Fatalf("expected INVALID_INPUT, got %v", env["error"])
	}
}

func TestSearchLocal_UnknownMode(t *testing.T) {
	tmp := t.TempDir()
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta", `export class Player {}
`)
	env := runLocalError(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Player",
		Mode:        "frob",
	})
	if env["error"].(map[string]any)["code"] != "INVALID_INPUT" {
		t.Fatalf("expected INVALID_INPUT, got %v", env["error"])
	}
}

func TestSearchLocal_NoMatch(t *testing.T) {
	tmp := t.TempDir()
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta", `export class Player {}
`)
	env := runLocalError(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "definitelyNotInFile",
		Mode:        "members",
	})
	if env["error"].(map[string]any)["code"] != "NO_MATCH" {
		t.Fatalf("expected NO_MATCH, got %v", env["error"])
	}
}

func TestSearchLocal_AllModulesSkipped(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "node_modules", "@minecraft", "server")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"@minecraft/server","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// No .d.ts file — module should be skipped.

	resp, err := handleSearchAPILocal(SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Player",
		Mode:        "members",
	})
	if err != nil {
		t.Fatalf("handleSearchAPILocal: %v", err)
	}
	body := searchAPIText(t, resp)
	var env map[string]any
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	skipped := env["skipped_modules"].([]any)
	if len(skipped) != 1 {
		t.Fatalf("skipped_modules = %v", skipped)
	}
	if len(env["modules_searched"].([]any)) != 0 {
		t.Fatalf("modules_searched should be empty when all skipped")
	}
}

func TestSearchLocal_ViaHandleSearchAPI(t *testing.T) {
	tmp := t.TempDir()
	writeModule(t, tmp, "@minecraft/server", "2.7.0-beta", `export class Player {}
`)
	resp, err := handleSearchAPI(SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Player",
		Mode:        "members",
	}, nil)
	if err != nil {
		t.Fatalf("handleSearchAPI: %v", err)
	}
	body := searchAPIText(t, resp)
	if !strings.Contains(body, `"source": "local"`) {
		t.Fatalf("expected local source in %s", body)
	}
}

func TestSearchLocal_FallsBackToFirstDtsFile(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "node_modules", "@minecraft", "server")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"@minecraft/server","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// No index.d.ts, but a types.d.ts file exists.
	if err := os.WriteFile(filepath.Join(dir, "types.d.ts"), []byte(`export class Widget {}
`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	env := runLocal(t, SearchAPIInput{
		Source:      "local",
		ProjectPath: tmp,
		Query:       "Widget",
		Mode:        "members",
	})
	if env["total_matches"].(float64) < 1 {
		t.Fatalf("fallback dts file not found: %v", env)
	}
}

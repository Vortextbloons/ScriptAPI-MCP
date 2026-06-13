package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleBPManifest = `{
  "format_version": 2,
  "header": {"name":"MyAddon","description":"d","uuid":"11111111-1111-1111-1111-111111111111","version":[1,0,0]},
  "modules": [{"type":"script","uuid":"22222222-2222-2222-2222-222222222222","version":[1,0,0],"language":"javascript","entry":"scripts/main.js"}]
}`

const sampleRPManifest = `{
  "format_version": 2,
  "header": {"name":"MyAddon RP","description":"d","uuid":"33333333-3333-3333-3333-333333333333","version":[1,0,0]},
  "modules": [{"type":"resources","uuid":"44444444-4444-4444-4444-444444444444","version":[1,0,0]}]
}`

func writeManifest(t *testing.T, root, packDir, body string) {
	t.Helper()
	dir := filepath.Join(root, packDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readName(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Header struct {
			Name string `json:"name"`
		} `json:"header"`
	}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m.Header.Name
}

func TestApplyDevSuffix(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		isDev  bool
		expect string
	}{
		{"adds suffix when missing", "MyAddon", true, "MyAddon-dev"},
		{"idempotent when already dev", "MyAddon-dev", true, "MyAddon-dev"},
		{"strips suffix when present", "MyAddon-dev", false, "MyAddon"},
		{"no-op when not dev and no suffix", "MyAddon", false, "MyAddon"},
		{"trims whitespace before checking", "  MyAddon  ", true, "MyAddon-dev"},
		{"empty becomes -dev when dev", "", true, "-dev"},
		{"empty stays empty when not dev", "", false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := applyDevSuffix(c.input, c.isDev)
			if got != c.expect {
				t.Errorf("applyDevSuffix(%q, %v) = %q, want %q", c.input, c.isDev, got, c.expect)
			}
		})
	}
}

func TestPrepareDevSuffix_NoManifests(t *testing.T) {
	root := t.TempDir()
	effective, report, restore, err := prepareDevSuffix(root, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if restore != nil {
		t.Fatal("expected nil restore when no manifests exist")
	}
	if report != nil {
		t.Fatalf("expected nil report when no manifests exist, got %+v", report)
	}
	if effective != root {
		t.Fatalf("expected unchanged project path, got %q", effective)
	}
}

func TestPrepareDevSuffix_EmptyNameNoOp(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "behavior_pack", `{"format_version":2,"header":{"name":"","description":"d","uuid":"11111111-1111-1111-1111-111111111111","version":[1,0,0]},"modules":[]}`)

	effective, report, restore, err := prepareDevSuffix(root, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if restore != nil || report != nil {
		t.Fatalf("expected nil restore/report, got restore=%p report=%+v", restore, report)
	}
	if effective != root {
		t.Fatalf("expected unchanged path, got %q", effective)
	}
}

func TestPrepareDevSuffix_ExplicitFalse_NoChange(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "behavior_pack", sampleBPManifest)
	writeManifest(t, root, "resource_pack", sampleRPManifest)

	effective, report, restore, err := prepareDevSuffix(root, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if restore != nil {
		restore()
	}

	if report == nil {
		t.Fatal("expected report")
	}
	if report.Requested != "non-dev" {
		t.Errorf("Requested = %q, want non-dev", report.Requested)
	}
	if report.BP == nil || report.BP.Changed || report.BP.Final != "MyAddon" {
		t.Errorf("BP entry unexpected: %+v", report.BP)
	}
	if report.RP == nil || report.RP.Changed || report.RP.Final != "MyAddon RP" {
		t.Errorf("RP entry unexpected: %+v", report.RP)
	}

	if effective != root {
		t.Fatalf("expected unchanged path when names already match non-dev, got %q", effective)
	}
}

func TestPrepareDevSuffix_ExplicitTrue_AlreadyDev(t *testing.T) {
	root := t.TempDir()
	body := strings.Replace(sampleBPManifest, `"name":"MyAddon"`, `"name":"MyAddon-dev"`, 1)
	writeManifest(t, root, "behavior_pack", body)

	effective, report, restore, err := prepareDevSuffix(root, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if restore != nil {
		restore()
	}

	if report == nil || report.Requested != "dev" {
		t.Fatalf("expected Requested=dev, got %+v", report)
	}
	if report.BP == nil || report.BP.Changed || report.BP.Final != "MyAddon-dev" {
		t.Errorf("BP entry unexpected: %+v", report.BP)
	}
	if report.RP != nil {
		t.Errorf("expected nil RP, got %+v", report.RP)
	}

	if effective != root {
		t.Fatalf("expected unchanged path when names already match dev, got %q", effective)
	}
}

func TestPrepareDevSuffix_ExplicitTrue_AddsSuffix(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "behavior_pack", sampleBPManifest)
	writeManifest(t, root, "resource_pack", sampleRPManifest)

	effective, report, restore, err := prepareDevSuffix(root, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer restore()

	if report == nil {
		t.Fatal("expected report")
	}
	if report.Requested != "dev" {
		t.Errorf("Requested = %q, want dev", report.Requested)
	}
	if report.BP == nil || !report.BP.Changed || report.BP.Final != "MyAddon-dev" {
		t.Errorf("BP unexpected: %+v", report.BP)
	}
	if report.RP == nil || !report.RP.Changed || report.RP.Final != "MyAddon RP-dev" {
		t.Errorf("RP unexpected: %+v", report.RP)
	}

	if effective == root {
		t.Fatal("expected staging path")
	}

	// Source manifest should be untouched.
	if name := readName(t, filepath.Join(root, "behavior_pack", "manifest.json")); name != "MyAddon" {
		t.Errorf("source BP name mutated: %q", name)
	}
	if name := readName(t, filepath.Join(root, "resource_pack", "manifest.json")); name != "MyAddon RP" {
		t.Errorf("source RP name mutated: %q", name)
	}

	// Staging manifests should reflect the patch.
	if name := readName(t, filepath.Join(effective, "behavior_pack", "manifest.json")); name != "MyAddon-dev" {
		t.Errorf("staged BP name: %q", name)
	}
	if name := readName(t, filepath.Join(effective, "resource_pack", "manifest.json")); name != "MyAddon RP-dev" {
		t.Errorf("staged RP name: %q", name)
	}
}

func TestPrepareDevSuffix_ExplicitFalse_StripsSuffix(t *testing.T) {
	root := t.TempDir()
	bpBody := strings.Replace(sampleBPManifest, `"name":"MyAddon"`, `"name":"MyAddon-dev"`, 1)
	rpBody := strings.Replace(sampleRPManifest, `"name":"MyAddon RP"`, `"name":"MyAddon RP-dev"`, 1)
	writeManifest(t, root, "behavior_pack", bpBody)
	writeManifest(t, root, "resource_pack", rpBody)

	effective, report, restore, err := prepareDevSuffix(root, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer restore()

	if report == nil || report.Requested != "non-dev" {
		t.Errorf("report unexpected: %+v", report)
	}
	if report.BP == nil || !report.BP.Changed || report.BP.Final != "MyAddon" {
		t.Errorf("BP unexpected: %+v", report.BP)
	}
	if report.RP == nil || !report.RP.Changed || report.RP.Final != "MyAddon RP" {
		t.Errorf("RP unexpected: %+v", report.RP)
	}

	if name := readName(t, filepath.Join(effective, "behavior_pack", "manifest.json")); name != "MyAddon" {
		t.Errorf("staged BP name: %q", name)
	}
	if name := readName(t, filepath.Join(effective, "resource_pack", "manifest.json")); name != "MyAddon RP" {
		t.Errorf("staged RP name: %q", name)
	}
}

func TestPrepareDevSuffix_IdempotentSkip(t *testing.T) {
	root := t.TempDir()
	bpBody := strings.Replace(sampleBPManifest, `"name":"MyAddon"`, `"name":"MyAddon-dev"`, 1)
	writeManifest(t, root, "behavior_pack", bpBody)

	effective, report, restore, err := prepareDevSuffix(root, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if restore != nil {
		restore()
		t.Fatal("expected nil restore when nothing changes")
	}
	if effective != root {
		t.Fatalf("expected unchanged path on idempotent skip, got %q", effective)
	}
	if report == nil || report.BP == nil || report.BP.Changed {
		t.Fatalf("expected BP entry with changed=false, got %+v", report)
	}
}

func TestPrepareDevSuffix_DryRunDoesNotStage(t *testing.T) {
	root := t.TempDir()
	writeManifest(t, root, "behavior_pack", sampleBPManifest)

	effective, report, restore, err := prepareDevSuffix(root, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if restore != nil {
		t.Fatal("expected nil restore on dry-run")
	}
	if effective != root {
		t.Fatalf("expected original path on dry-run, got %q", effective)
	}
	if report == nil || report.BP == nil || !report.BP.Changed || report.BP.Final != "MyAddon-dev" {
		t.Errorf("expected report to show planned final name, got %+v", report)
	}
}

func TestApplyDevSuffixToOutputPath(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		isDev  bool
		expect string
	}{
		{"production strips suffix from mcaddon name", `C:\out\MyAddon-dev.mcaddon`, false, `C:\out\MyAddon.mcaddon`},
		{"dev adds suffix to mcaddon name", `C:\out\MyAddon.mcaddon`, true, `C:\out\MyAddon-dev.mcaddon`},
		{"idempotent dev mcaddon path", `C:\out\MyAddon-dev.mcaddon`, true, `C:\out\MyAddon-dev.mcaddon`},
		{"idempotent production mcaddon path", `C:\out\MyAddon.mcaddon`, false, `C:\out\MyAddon.mcaddon`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := applyDevSuffixToOutputPath(c.input, c.isDev)
			if got != c.expect {
				t.Errorf("applyDevSuffixToOutputPath(%q, %v) = %q, want %q", c.input, c.isDev, got, c.expect)
			}
		})
	}
}

func TestPrepareDevSuffix_InvalidManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "behavior_pack"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "behavior_pack", "manifest.json"), []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, _, err := prepareDevSuffix(root, true, false)
	if err == nil {
		t.Fatal("expected error for invalid manifest JSON")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("expected parse error, got: %v", err)
	}
}


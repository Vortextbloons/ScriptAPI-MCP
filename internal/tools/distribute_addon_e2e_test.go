package tools

import (
	"strings"
	"testing"
)

func TestE2E_TauGemUpgrades_Production(t *testing.T) {
	projectPath := `C:\Users\isaac\Desktop\DevProjects\Minecraft & Tau\Tau Gem Upgrades`
	if !fileExists(projectPath) {
		t.Skip("user project not present on this machine")
	}

	layout, err := resolvePackLayout(projectPath, "", "", "")
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	if layout.BPSource != "static/bp" || layout.RPSource != "static/rp" {
		t.Fatalf("expected static layout, got %+v", layout)
	}
	if layout.ScriptsSource != "dist" {
		t.Fatalf("expected scripts_source=dist, got %q", layout.ScriptsSource)
	}

	effective, report, restore, err := prepareDevSuffix(projectPath, layout, false, false)
	if err != nil {
		t.Fatalf("prepareDevSuffix: %v", err)
	}
	defer restore()

	if report == nil || report.BP == nil || report.RP == nil {
		t.Fatalf("expected BP and RP entries, got %+v", report)
	}
	if !strings.HasSuffix(report.BP.Original, "-dev") {
		t.Errorf("expected source BP to end with -dev, got %q", report.BP.Original)
	}
	if strings.HasSuffix(report.BP.Final, "-dev") {
		t.Errorf("expected production BP to NOT end with -dev, got %q", report.BP.Final)
	}
	if !strings.HasSuffix(report.RP.Original, "-dev") {
		t.Errorf("expected source RP to end with -dev, got %q", report.RP.Original)
	}
	if strings.HasSuffix(report.RP.Final, "-dev") {
		t.Errorf("expected production RP to NOT end with -dev, got %q", report.RP.Final)
	}

	// Source untouched.
	if name := readName(t, projectPath+`\static\bp\manifest.json`); !strings.HasSuffix(name, "-dev") {
		t.Errorf("source BP name should retain -dev, got %q", name)
	}
	// Staging copy has the patched names.
	if name := readName(t, effective+`\static\bp\manifest.json`); strings.HasSuffix(name, "-dev") {
		t.Errorf("staged BP name should have -dev stripped, got %q", name)
	}

	out := t.TempDir() + `\Tau Gem Upgrades.mcaddon`
	pkg, err := packageAddon(PackageAddonInput{
		ProjectPath:   effective,
		OutputPath:    out,
		BPSource:      layout.BPSource,
		RPSource:      layout.RPSource,
		ScriptsSource: layout.ScriptsSource,
	})
	if err != nil {
		t.Fatalf("packageAddon: %v", err)
	}
	if pkg.BPIncluded == 0 || pkg.RPIncluded == 0 || pkg.ScriptsIncluded == 0 {
		t.Errorf("expected all three sources to contribute, got %+v", pkg)
	}
	if pkg.OutputPath == "" {
		t.Error("expected output path to be set")
	}

	entries := readZipEntries(t, pkg.OutputPath)
	want := []string{
		"static/bp/manifest.json",
		"static/rp/manifest.json",
		"scripts/index.js",
	}
	for _, w := range want {
		found := false
		for _, e := range entries {
			if e == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in zip, got %v", w, entries)
		}
	}
}

func TestE2E_TauGemUpgrades_Dev(t *testing.T) {
	projectPath := `C:\Users\isaac\Desktop\DevProjects\Minecraft & Tau\Tau Gem Upgrades`
	if !fileExists(projectPath) {
		t.Skip("user project not present on this machine")
	}

	layout, err := resolvePackLayout(projectPath, "", "", "")
	if err != nil {
		t.Fatalf("layout: %v", err)
	}

	effective, report, restore, err := prepareDevSuffix(projectPath, layout, true, false)
	if err != nil {
		t.Fatalf("prepareDevSuffix: %v", err)
	}
	if restore != nil {
		defer restore()
	}
	_ = effective

	if report == nil || report.BP == nil {
		t.Fatalf("expected BP entry, got %+v", report)
	}
	if !strings.HasSuffix(report.BP.Final, "-dev") {
		t.Errorf("expected dev BP to end with -dev, got %q", report.BP.Final)
	}
	if !report.BP.Changed && !strings.HasSuffix(report.BP.Original, "-dev") {
		t.Errorf("expected BP to be marked changed or already have -dev, got %+v", report.BP)
	}
}

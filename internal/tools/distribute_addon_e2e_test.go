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

	// Top-level pack names default to the (suffix-stripped) manifest
	// header.name so the .mcaddon has friendly folders.
	bpPackName := report.BP.Final // "Tau Gem Upgrades BP 2.8.3-Beta"
	rpPackName := report.RP.Final // "Tau Gem Upgrades RP 2.8.3-Beta"

	out := t.TempDir() + `\Tau Gem Upgrades.mcaddon`
	pkg, err := packageAddon(PackageAddonInput{
		ProjectPath:   effective,
		OutputPath:    out,
		BPSource:      layout.BPSource,
		RPSource:      layout.RPSource,
		ScriptsSource: layout.ScriptsSource,
		BPPackName:    bpPackName,
		RPPackName:    rpPackName,
	})
	if err != nil {
		t.Fatalf("packageAddon: %v", err)
	}
	if pkg.BPIncluded == 0 || pkg.RPIncluded == 0 || pkg.ScriptsIncluded == 0 {
		t.Errorf("expected all three sources to contribute, got %+v", pkg)
	}

	entries := readZipEntries(t, pkg.OutputPath)
	// The .mcaddon must have two top-level pack folders, with the BP
	// containing the manifest at the root (no nested "static/bp/" prefix)
	// and scripts/ inside the BP.
	want := []string{
		bpPackName + "/manifest.json",
		rpPackName + "/manifest.json",
		bpPackName + "/scripts/index.js",
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

	// And the .mcaddon must NOT contain the source-path layout.
	for _, bad := range []string{"static/bp/manifest.json", "static/rp/manifest.json"} {
		for _, e := range entries {
			if e == bad {
				t.Errorf("zip should not contain source-path layout entry %q", bad)
			}
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

	_, report, restore, err := prepareDevSuffix(projectPath, layout, true, false)
	if err != nil {
		t.Fatalf("prepareDevSuffix: %v", err)
	}
	if restore != nil {
		defer restore()
	}

	if report == nil || report.BP == nil {
		t.Fatalf("expected BP entry, got %+v", report)
	}
	if !strings.HasSuffix(report.BP.Final, "-dev") {
		t.Errorf("expected dev BP to end with -dev, got %q", report.BP.Final)
	}
}

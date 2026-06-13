package tools

import (
	"archive/zip"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestE2E_TauGemUpgrades_ZipShape(t *testing.T) {
	projectPath := `C:\Users\isaac\Desktop\DevProjects\Minecraft & Tau\Tau Gem Upgrades`
	if !fileExists(projectPath) {
		t.Skip("user project not present on this machine")
	}

	layout, err := resolvePackLayout(projectPath, "", "", "")
	if err != nil {
		t.Fatalf("layout: %v", err)
	}
	effective, _, restore, err := prepareDevSuffix(projectPath, layout, false, false)
	if err != nil {
		t.Fatalf("prepareDevSuffix: %v", err)
	}
	defer restore()

	out := t.TempDir() + `\Tau Gem Upgrades.mcaddon`
	pkg, err := packageAddon(PackageAddonInput{
		ProjectPath:   effective,
		OutputPath:    out,
		BPSource:      layout.BPSource,
		RPSource:      layout.RPSource,
		ScriptsSource: layout.ScriptsSource,
		BPPackName:    "Tau Gem Upgrades BP",
		RPPackName:    "Tau Gem Upgrades RP",
	})
	if err != nil {
		t.Fatalf("packageAddon: %v", err)
	}

	// Print the unique top-level folder names so the test output shows
	// whether the .mcaddon has the right shape.
	r, err := zip.OpenReader(pkg.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	tops := map[string]int{}
	hasManifestAtBP := false
	hasManifestAtRP := false
	hasScriptsInBP := false
	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		parts := strings.SplitN(name, "/", 2)
		if len(parts) >= 1 {
			tops[parts[0]]++
		}
		switch {
		case name == "Tau Gem Upgrades BP/manifest.json":
			hasManifestAtBP = true
		case name == "Tau Gem Upgrades RP/manifest.json":
			hasManifestAtRP = true
		case strings.HasPrefix(name, "Tau Gem Upgrades BP/scripts/"):
			hasScriptsInBP = true
		}
	}

	keys := make([]string, 0, len(tops))
	for k := range tops {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("top-level folders in zip: %v (counts: %v)", keys, tops)
	t.Logf("rewrote: %v", pkg.RewroteLayout)

	if !hasManifestAtBP {
		t.Error("BP manifest not at Tau Gem Upgrades BP/manifest.json")
	}
	if !hasManifestAtRP {
		t.Error("RP manifest not at Tau Gem Upgrades RP/manifest.json")
	}
	if !hasScriptsInBP {
		t.Error("no scripts/ entries under Tau Gem Upgrades BP/")
	}
	if _, ok := tops["static"]; ok {
		t.Errorf("zip should not have a 'static' top-level folder")
	}
}

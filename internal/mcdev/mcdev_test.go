package mcdev

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withEnv(t *testing.T, key, value string) {
	t.Helper()
	oldGet := envGet
	t.Cleanup(func() { envGet = oldGet })
	envGet = func(k string) string {
		if k == key {
			return value
		}
		return ""
	}
}

func withPlatform(t *testing.T, name string, home string, appData string, localApp string, globs map[string][]string) {
	t.Helper()
	oldGoos, oldHome, oldGet, oldGlob := goos, userHomeDir, envGet, globFunc
	t.Cleanup(func() { goos, userHomeDir, envGet, globFunc = oldGoos, oldHome, oldGet, oldGlob })
	goos = name
	userHomeDir = func() (string, error) { return home, nil }
	envGet = func(k string) string {
		switch k {
		case "APPDATA":
			return appData
		case "LOCALAPPDATA":
			return localApp
		case EnvPrimary, EnvAlias:
			return ""
		}
		return ""
	}
	globFunc = func(pattern string) ([]string, error) {
		if v, ok := globs[pattern]; ok {
			return v, nil
		}
		return nil, nil
	}
}

func TestExplicitWinsOverEnv(t *testing.T) {
	oldGet := envGet
	defer func() { envGet = oldGet }()
	envGet = func(k string) string {
		if k == EnvPrimary {
			return `C:\from-env`
		}
		return ""
	}
	res, err := Resolve(`C:\explicit`)
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != "explicit" || res.Path != `C:\explicit` {
		t.Fatalf("got %+v", res)
	}
	if !strings.HasSuffix(res.DevBP, "development_behavior_packs") {
		t.Fatalf("DevBP = %q", res.DevBP)
	}
}

func TestEnvPrimaryWins(t *testing.T) {
	withEnv(t, EnvPrimary, `/tmp/mc`)
	res, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != "env" {
		t.Fatalf("source = %q", res.Source)
	}
}

func TestEnvAliasFallback(t *testing.T) {
	oldGet := envGet
	defer func() { envGet = oldGet }()
	envGet = func(k string) string {
		if k == EnvAlias {
			return `/tmp/alias`
		}
		return ""
	}
	res, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != "env" || res.Path != `/tmp/alias` {
		t.Fatalf("got %+v", res)
	}
}

func TestWindowsLegacyCandidate(t *testing.T) {
	root := t.TempDir()
	comMojang := filepath.Join(root, "Minecraft Bedrock", "Users", "Shared", "games", "com.mojang")
	if err := os.MkdirAll(filepath.Join(comMojang, "development_behavior_packs"), 0o755); err != nil {
		t.Fatal(err)
	}
	withPlatform(t, "windows", "", root, "", nil)
	res, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != "detected" || res.Path != comMojang {
		t.Fatalf("got %+v", res)
	}
}

func TestWindowsUWPCandidate(t *testing.T) {
	root := t.TempDir()
	uwp := filepath.Join(root, "Packages", "Microsoft.MinecraftUWP_8wekyb3d8bbwe", "LocalState", "games", "com.mojang")
	if err := os.MkdirAll(filepath.Join(uwp, "behavior_packs"), 0o755); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(root, "Packages", "Microsoft.*8wekyb3d8bbwe", "LocalState", "games", "com.mojang")
	withPlatform(t, "windows", "", `C:\nonexistent-appdata-xyz`, root, map[string][]string{pattern: {uwp}})
	res, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != uwp {
		t.Fatalf("got %+v", res)
	}
}

func TestDarwinCandidate(t *testing.T) {
	home := t.TempDir()
	mcpe := filepath.Join(home, "Library", "Application Support", "minecraftpe")
	if err := os.MkdirAll(filepath.Join(mcpe, "resource_packs"), 0o755); err != nil {
		t.Fatal(err)
	}
	withPlatform(t, "darwin", home, "", "", nil)
	res, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != mcpe {
		t.Fatalf("got %+v", res)
	}
}

func TestLinuxCandidates(t *testing.T) {
	home := t.TempDir()
	mcpe := filepath.Join(home, ".local", "share", "minecraftpe")
	if err := os.MkdirAll(filepath.Join(mcpe, "development_behavior_packs"), 0o755); err != nil {
		t.Fatal(err)
	}
	withPlatform(t, "linux", home, "", "", nil)
	res, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != mcpe {
		t.Fatalf("got %+v", res)
	}
}

func TestNotFoundListsProbes(t *testing.T) {
	withPlatform(t, "windows", "", `C:\no-appdata`, `C:\no-localappdata`, nil)
	_, err := Resolve("")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), EnvPrimary) {
		t.Fatalf("error should mention env var, got: %v", err)
	}
}

func TestEnsureDevDirs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "com.mojang")
	if err := EnsureDevDirs(root); err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{"development_behavior_packs", "development_resource_packs"} {
		info, err := os.Stat(filepath.Join(root, sub))
		if err != nil || !info.IsDir() {
			t.Fatalf("missing %s: %v", sub, err)
		}
	}
}

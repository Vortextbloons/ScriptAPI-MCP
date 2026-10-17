package mcdev

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Env vars, in priority order. MINECRAFT_COM_MOJANG is canonical;
// MCDEV_PATH is accepted as an alias.
const (
	EnvPrimary = "MINECRAFT_COM_MOJANG"
	EnvAlias   = "MCDEV_PATH"
)

// Resolution describes where Bedrock deploy targets live.
type Resolution struct {
	Path       string   `json:"path"`
	Source     string   `json:"source"`
	DevBP      string   `json:"dev_behavior_packs"`
	DevRP      string   `json:"dev_resource_packs"`
	Exists     bool     `json:"exists"`
	Platform   string   `json:"platform"`
	Probed     []string `json:"candidates_probed,omitempty"`
}

// overridable for tests
var (
	envGet      = os.Getenv
	goos        = runtime.GOOS
	globFunc    = filepath.Glob
	statFunc    = os.Stat
	userHomeDir = os.UserHomeDir
)

// Resolve returns the games data root that holds the pack folders.
// Precedence: explicit arg > MINECRAFT_COM_MOJANG/MCDEV_PATH env > platform detection.
func Resolve(explicit string) (Resolution, error) {
	if trimmed := strings.TrimSpace(explicit); trimmed != "" {
		abs := expandPath(trimmed)
		return Resolution{
			Path:     abs,
			Source:   "explicit",
			DevBP:    filepath.Join(abs, "development_behavior_packs"),
			DevRP:    filepath.Join(abs, "development_resource_packs"),
			Exists:   dirExists(abs),
			Platform: goos,
		}, nil
	}

	if v := strings.TrimSpace(envGet(EnvPrimary)); v != "" {
		abs := expandPath(v)
		return Resolution{
			Path:     abs,
			Source:   "env",
			DevBP:    filepath.Join(abs, "development_behavior_packs"),
			DevRP:    filepath.Join(abs, "development_resource_packs"),
			Exists:   dirExists(abs),
			Platform: goos,
		}, nil
	}
	if v := strings.TrimSpace(envGet(EnvAlias)); v != "" {
		abs := expandPath(v)
		return Resolution{
			Path:     abs,
			Source:   "env",
			DevBP:    filepath.Join(abs, "development_behavior_packs"),
			DevRP:    filepath.Join(abs, "development_resource_packs"),
			Exists:   dirExists(abs),
			Platform: goos,
		}, nil
	}

	probed := platformCandidates()
	for _, c := range probed {
		if isGamesRoot(c) {
			return Resolution{
				Path:     c,
				Source:   "detected",
				DevBP:    filepath.Join(c, "development_behavior_packs"),
				DevRP:    filepath.Join(c, "development_resource_packs"),
				Exists:   true,
				Platform: goos,
				Probed:   probed,
			}, nil
		}
	}
	return Resolution{Platform: goos, Probed: probed}, fmt.Errorf(
		"could not locate Minecraft games data folder; checked: %s; set %s (or %s) to the folder containing development_behavior_packs",
		strings.Join(probed, "; "), EnvPrimary, EnvAlias)
}

// EnsureDevDirs creates development_behavior_packs and development_resource_packs
// under root. It is a no-op for empty roots.
func EnsureDevDirs(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("empty games data root")
	}
	for _, sub := range []string{"development_behavior_packs", "development_resource_packs"} {
		if err := os.MkdirAll(filepath.Join(root, sub), 0o755); err != nil {
			return err
		}
	}
	return nil
}

// platformCandidates returns the detection chain for the current OS.
func platformCandidates() []string {
	out := []string{}
	switch goos {
	case "windows":
		if appData := strings.TrimSpace(envGet("APPDATA")); appData != "" {
			out = append(out, filepath.Join(appData, "Minecraft Bedrock", "Users", "Shared", "games", "com.mojang"))
		}
		if localApp := strings.TrimSpace(envGet("LOCALAPPDATA")); localApp != "" {
			matches, _ := globFunc(filepath.Join(localApp, "Packages", "Microsoft.*8wekyb3d8bbwe", "LocalState", "games", "com.mojang"))
			out = append(out, matches...)
		}
	case "darwin":
		if home, err := userHomeDir(); err == nil && home != "" {
			out = append(out, filepath.Join(home, "Library", "Application Support", "minecraftpe"))
		}
	default: // linux and others
		if home, err := userHomeDir(); err == nil && home != "" {
			out = append(out,
				filepath.Join(home, ".local", "share", "minecraftpe"),
				filepath.Join(home, ".var", "app", "com.mojang.Minecraft", "data", "minecraftpe"),
			)
		}
	}
	return out
}

// isGamesRoot reports whether dir exists and looks like a games data root
// (holds pack folders).
func isGamesRoot(dir string) bool {
	if !dirExists(dir) {
		return false
	}
	for _, marker := range []string{
		"development_behavior_packs",
		"development_resource_packs",
		"behavior_packs",
		"resource_packs",
	} {
		if dirExists(filepath.Join(dir, marker)) {
			return true
		}
	}
	return false
}

func dirExists(path string) bool {
	info, err := statFunc(path)
	return err == nil && info.IsDir()
}

func expandPath(p string) string {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, "~") {
		if home, err := userHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			rest := strings.TrimPrefix(p[1:], `\`)
			rest = strings.TrimPrefix(rest, "/")
			return filepath.Join(home, rest)
		}
	}
	return os.ExpandEnv(p)
}

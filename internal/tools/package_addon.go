package tools

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type PackageAddonInput struct {
	ProjectPath   string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
	OutputPath    string `json:"output_path" mcp:"required,description='Path for .mcaddon output file'"`
	BPSource      string `json:"bp_source" mcp:"description='BP source folder relative to project_path (auto-detected if empty)'"`
	RPSource      string `json:"rp_source" mcp:"description='RP source folder relative to project_path (auto-detected if empty)'"`
	ScriptsSource string `json:"scripts_source" mcp:"description='Folder whose contents are merged into the BP at scripts/ during packaging (e.g. dist). Skipped if empty or not found.'"`
	BPPackName    string `json:"bp_pack_name" mcp:"description='Top-level folder name written into the .mcaddon for the BP (default: behavior_pack). For Bedrock loaders, set this to a friendly name like \"Tau Gem Upgrades BP\".'"`
	RPPackName    string `json:"rp_pack_name" mcp:"description='Top-level folder name written into the .mcaddon for the RP (default: resource_pack).'"`
	KeepLayout    bool   `json:"keep_layout" mcp:"description='If true, keep the source folder layout inside the .mcaddon (e.g. static/bp/... at root). Default false rewrites the top-level folder to <bp_pack_name>/<rp_pack_name>, which is what Bedrock expects.'"`
	DryRun        bool   `json:"dry_run" mcp:"description='If true, previews packaging without writing file'"`
}

type PackageAddonOutput struct {
	OutputPath      string   `json:"output_path"`
	Included        []string `json:"included"`
	BPIncluded      int      `json:"bp_included"`
	RPIncluded      int      `json:"rp_included"`
	ScriptsIncluded int      `json:"scripts_included"`
	BPPackName      string   `json:"bp_pack_name"`
	RPPackName      string   `json:"rp_pack_name"`
	RewroteLayout   bool     `json:"rewrote_layout"`
	DryRun          bool     `json:"dry_run"`
}

// zipEntry is one file to write into the .mcaddon archive. srcAbs is the
// on-disk path to read from; archiveName is the path written inside the zip.
type zipEntry struct {
	srcAbs       string
	archiveName string
	// packTag records which logical group this entry belongs to: "bp",
	// "rp", or "scripts". The top-level rewrite uses this to route each
	// entry under the right pack folder.
	packTag string
}

func packageAddon(args PackageAddonInput) (*PackageAddonOutput, error) {
	projectPath := strings.TrimSpace(args.ProjectPath)
	outputPath := strings.TrimSpace(args.OutputPath)
	if projectPath == "" || outputPath == "" {
		return nil, fmt.Errorf("project_path and output_path are required")
	}

	bpPackName := strings.TrimSpace(args.BPPackName)
	if bpPackName == "" {
		bpPackName = "behavior_pack"
	}
	rpPackName := strings.TrimSpace(args.RPPackName)
	if rpPackName == "" {
		rpPackName = "resource_pack"
	}

	entries := make([]zipEntry, 0)

	// When KeepLayout is true, the project-relative path is used so the
	// source folder structure is preserved verbatim in the archive.
	// Otherwise the source-relative path is used and the top-level
	// rewrite prepends the pack name.
	bpRel := strings.TrimSpace(args.BPSource)
	if bpRel == "" {
		bpRel = "behavior_pack"
	}
	rpRel := strings.TrimSpace(args.RPSource)
	if rpRel == "" {
		rpRel = "resource_pack"
	}
	scriptsRel := strings.TrimSpace(args.ScriptsSource)

	bpCount := 0
	if abs, ok := safeSubdir(projectPath, bpRel); ok {
		base := abs
		if args.KeepLayout {
			base = projectPath
		}
		count, err := walkForZip(abs, base, &entries, "", "bp")
		if err != nil {
			return nil, err
		}
		bpCount = count
	}

	rpCount := 0
	if abs, ok := safeSubdir(projectPath, rpRel); ok {
		base := abs
		if args.KeepLayout {
			base = projectPath
		}
		count, err := walkForZip(abs, base, &entries, "", "rp")
		if err != nil {
			return nil, err
		}
		rpCount = count
	}

	scriptsCount := 0
	if scriptsRel != "" {
		if abs, ok := safeSubdir(projectPath, scriptsRel); ok {
			base := abs
			if args.KeepLayout {
				base = projectPath
			}
			count, err := walkForZip(abs, base, &entries, "", "scripts")
			if err != nil {
				return nil, err
			}
			scriptsCount = count
		}
	}

	// Detect whether a layout rewrite is actually needed. The rewrite
	// prepends the pack name to every entry; it's a no-op in terms of
	// path shape when the source folder's basename already matches the
	// pack name (e.g. behavior_pack/ + behavior_pack). For non-standard
	// sources (static/bp, src/rp, etc.) or when the user has set
	// BPPackName/RPPackName to a custom name, rewrote is true so the
	// response can flag the transformation.
	rewrote := !args.KeepLayout && (sourceBasename(bpRel) != bpPackName || sourceBasename(rpRel) != rpPackName)
	if !args.KeepLayout {
		for i := range entries {
			entries[i].archiveName = applyPackName(entries[i].archiveName, entries[i].packTag, bpPackName, rpPackName)
		}
	}

	included := make([]string, 0, len(entries))
	for _, e := range entries {
		included = append(included, e.archiveName)
	}

	if args.DryRun {
		return &PackageAddonOutput{
			OutputPath:      outputPath,
			Included:        included,
			BPIncluded:      bpCount,
			RPIncluded:      rpCount,
			ScriptsIncluded: scriptsCount,
			BPPackName:      bpPackName,
			RPPackName:      rpPackName,
			RewroteLayout:   rewrote,
			DryRun:          true,
		}, nil
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, e := range entries {
		in, err := os.Open(e.srcAbs)
		if err != nil {
			return nil, err
		}
		w, err := zw.Create(filepath.ToSlash(e.archiveName))
		if err != nil {
			in.Close()
			return nil, err
		}
		_, err = io.Copy(w, in)
		in.Close()
		if err != nil {
			return nil, err
		}
	}

	return &PackageAddonOutput{
		OutputPath:      outputPath,
		Included:        included,
		BPIncluded:      bpCount,
		RPIncluded:      rpCount,
		ScriptsIncluded: scriptsCount,
		BPPackName:      bpPackName,
		RPPackName:      rpPackName,
		RewroteLayout:   rewrote,
		DryRun:          false,
	}, nil
}

// applyPackName prepends the matching pack-name folder to the archive
// path. archiveName is expected to be relative to the pack source root
// (e.g. "manifest.json" or "blocks/foo.json"), with no leading prefix.
func applyPackName(archiveName, packTag, bpPackName, rpPackName string) string {
	switch packTag {
	case "bp":
		if archiveName == "" {
			return bpPackName
		}
		return bpPackName + "/" + archiveName
	case "rp":
		if archiveName == "" {
			return rpPackName
		}
		return rpPackName + "/" + archiveName
	case "scripts":
		if archiveName == "" {
			return bpPackName + "/scripts"
		}
		return bpPackName + "/scripts/" + archiveName
	}
	return archiveName
}

// sourceBasename returns the last segment of a slash-separated path, used
// to compare a source folder (e.g. "static/bp") against the chosen pack
// name (e.g. "behavior_pack") so we can flag when a rewrite is needed.
func sourceBasename(rel string) string {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	return parts[len(parts)-1]
}

// safeSubdir joins projectPath and rel, returning the absolute path and true
// when the resulting directory exists. It blocks ".." traversal.
func safeSubdir(projectPath, rel string) (string, bool) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", false
	}
	if strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) {
		return "", false
	}
	cleaned := filepath.Clean(filepath.FromSlash(rel))
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", false
	}
	abs := filepath.Join(projectPath, cleaned)
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return abs, true
}

// walkForZip walks absRoot and appends one zipEntry per file. relBase is
// the directory the archiveName is computed relative to:
//   - When relBase == absRoot (default), archiveName is the source-relative
//     path (e.g. "manifest.json") and the top-level rewrite prepends the
//     pack name.
//   - When relBase is the project root, archiveName is the project-relative
//     path (e.g. "static/bp/manifest.json") and the top-level rewrite is
//     skipped under KeepLayout, preserving the source layout exactly.
//
// packTag labels the entry so the top-level rewrite can route it correctly.
func walkForZip(absRoot, relBase string, out *[]zipEntry, zipPrefix, packTag string) (int, error) {
	count := 0
	err := filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(relBase, path)
		if rerr != nil {
			return rerr
		}
		archiveName := filepath.ToSlash(rel)
		if zipPrefix != "" {
			archiveName = filepath.ToSlash(filepath.Join(zipPrefix, rel))
		}
		*out = append(*out, zipEntry{srcAbs: path, archiveName: archiveName, packTag: packTag})
		count++
		return nil
	})
	return count, err
}

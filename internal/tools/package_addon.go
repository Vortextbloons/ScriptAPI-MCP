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
	DryRun        bool   `json:"dry_run" mcp:"description='If true, previews packaging without writing file'"`
}

type PackageAddonOutput struct {
	OutputPath      string   `json:"output_path"`
	Included        []string `json:"included"`
	BPIncluded      int      `json:"bp_included"`
	RPIncluded      int      `json:"rp_included"`
	ScriptsIncluded int      `json:"scripts_included"`
	DryRun          bool     `json:"dry_run"`
}

// zipEntry is one file to write into the .mcaddon archive. srcAbs is the
// on-disk path to read from; archiveName is the path written inside the zip.
type zipEntry struct {
	srcAbs       string
	archiveName string
}

func packageAddon(args PackageAddonInput) (*PackageAddonOutput, error) {
	projectPath := strings.TrimSpace(args.ProjectPath)
	outputPath := strings.TrimSpace(args.OutputPath)
	if projectPath == "" || outputPath == "" {
		return nil, fmt.Errorf("project_path and output_path are required")
	}

	entries := make([]zipEntry, 0)

	bpRel := strings.TrimSpace(args.BPSource)
	if bpRel == "" {
		bpRel = "behavior_pack"
	}
	bpCount := 0
	if abs, ok := safeSubdir(projectPath, bpRel); ok {
		count, err := walkForZip(abs, projectPath, &entries, "")
		if err != nil {
			return nil, err
		}
		bpCount = count
	}

	rpRel := strings.TrimSpace(args.RPSource)
	if rpRel == "" {
		rpRel = "resource_pack"
	}
	rpCount := 0
	if abs, ok := safeSubdir(projectPath, rpRel); ok {
		count, err := walkForZip(abs, projectPath, &entries, "")
		if err != nil {
			return nil, err
		}
		rpCount = count
	}

	scriptsCount := 0
	scriptsRel := strings.TrimSpace(args.ScriptsSource)
	if scriptsRel != "" {
		if abs, ok := safeSubdir(projectPath, scriptsRel); ok {
			count, err := walkForZip(abs, abs, &entries, "scripts")
			if err != nil {
				return nil, err
			}
			scriptsCount = count
		}
	}

	// Build the included []string for the dry-run report.
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
		DryRun:          false,
	}, nil
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

// walkForZip walks absRoot and appends one zipEntry per file. The
// archiveName is computed as filepath.Rel(relBase, path) with optional
// zipPrefix prepended.
func walkForZip(absRoot, relBase string, out *[]zipEntry, zipPrefix string) (int, error) {
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
		*out = append(*out, zipEntry{srcAbs: path, archiveName: archiveName})
		count++
		return nil
	})
	return count, err
}

package tools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/mcdev"
)

type DeployAddonInput struct {
	ProjectPath   string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
	MCDevPath     string `json:"mcdev_path" mcp:"description='Path to com.mojang / games data folder. Optional: resolved from MINECRAFT_COM_MOJANG env or platform auto-detection when empty.'"`
	BPSource      string `json:"bp_source" mcp:"description='BP source folder relative to project_path (auto-detected if empty)'"`
	RPSource      string `json:"rp_source" mcp:"description='RP source folder relative to project_path (auto-detected if empty)'"`
	ScriptsSource string `json:"scripts_source" mcp:"description='Folder whose contents are merged into the BP at scripts/ during deploy (e.g. dist). Skipped if empty or not found.'"`
	BPDestName    string `json:"bp_dest_name" mcp:"description='Destination behavior pack folder name'"`
	RPDestName    string `json:"rp_dest_name" mcp:"description='Destination resource pack folder name'"`
	DryRun        bool   `json:"dry_run" mcp:"description='If true, previews deploy without writing'"`
}

type DeployAddonOutput struct {
	Operations      []string `json:"operations"`
	BPSource        string   `json:"bp_source"`
	RPSource        string   `json:"rp_source"`
	MCDevPath       string   `json:"mcdev_path"`
	MCDevSource     string   `json:"mcdev_source,omitempty"`
	DryRun          bool     `json:"dry_run"`
}

func deployAddon(args DeployAddonInput) (*DeployAddonOutput, error) {
	resolved, err := mcdev.Resolve(args.MCDevPath)
	if err != nil {
		return nil, err
	}
	mcdevPath := resolved.Path

	bpRel := strings.TrimSpace(args.BPSource)
	if bpRel == "" {
		bpRel = "behavior_pack"
	}
	rpRel := strings.TrimSpace(args.RPSource)
	if rpRel == "" {
		rpRel = "resource_pack"
	}

	bpSrc, bpOK := safeSubdir(args.ProjectPath, bpRel)
	if !bpOK {
		return nil, fmt.Errorf("behavior pack source folder not found: %s", bpRel)
	}
	bpName := strings.TrimSpace(args.BPDestName)
	if bpName == "" {
		bpName = "AddonBP"
	}
	rpName := strings.TrimSpace(args.RPDestName)
	if rpName == "" {
		rpName = "AddonRP"
	}
	rpSrc, rpOK := safeSubdir(args.ProjectPath, rpRel)

	bpDst := filepath.Join(mcdevPath, "development_behavior_packs", bpName)

	ops := []string{fmt.Sprintf("copy %s -> %s", bpSrc, bpDst)}
	if rpOK {
		rpDst := filepath.Join(mcdevPath, "development_resource_packs", rpName)
		ops = append(ops, fmt.Sprintf("copy %s -> %s", rpSrc, rpDst))
	}

	scriptsRel := strings.TrimSpace(args.ScriptsSource)
	if scriptsRel != "" {
		if scriptsAbs, ok := safeSubdir(args.ProjectPath, scriptsRel); ok {
			ops = append(ops, fmt.Sprintf("copy %s -> %s/scripts", scriptsAbs, bpDst))
		}
	}

	if args.DryRun {
		return &DeployAddonOutput{
			Operations:  ops,
			BPSource:    bpRel,
			RPSource:    rpRel,
			MCDevPath:   mcdevPath,
			MCDevSource: resolved.Source,
			DryRun:      true,
		}, nil
	}

	if err := mcdev.EnsureDevDirs(mcdevPath); err != nil {
		return nil, err
	}
	if err := copyDir(bpSrc, bpDst); err != nil {
		return nil, err
	}
	if rpOK {
		rpDst := filepath.Join(mcdevPath, "development_resource_packs", rpName)
		if err := copyDir(rpSrc, rpDst); err != nil {
			return nil, err
		}
	}
	if scriptsRel != "" {
		if scriptsAbs, ok := safeSubdir(args.ProjectPath, scriptsRel); ok {
			scriptsDst := filepath.Join(bpDst, "scripts")
			if err := copyDir(scriptsAbs, scriptsDst); err != nil {
				return nil, err
			}
		}
	}

	return &DeployAddonOutput{
		Operations:  ops,
		BPSource:    bpRel,
		RPSource:    rpRel,
		MCDevPath:   mcdevPath,
		MCDevSource: resolved.Source,
		DryRun:      false,
	}, nil
}

func copyDir(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

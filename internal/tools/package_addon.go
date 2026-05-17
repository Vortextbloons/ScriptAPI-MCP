package tools

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"
)

type PackageAddonInput struct {
	ProjectPath string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
	OutputPath  string `json:"output_path" mcp:"required,description='Path for .mcaddon output file'"`
	DryRun      bool   `json:"dry_run" mcp:"description='If true, previews packaging without writing file'"`
}

type PackageAddonOutput struct {
	OutputPath string   `json:"output_path"`
	Included   []string `json:"included"`
	DryRun     bool     `json:"dry_run"`
}

func RegisterPackageAddon(server *mcp.Server) error {
	return server.RegisterTool("package_addon",
		"Packages an addon workspace into a .mcaddon archive cross-platform.",
		func(args PackageAddonInput) (*mcp.ToolResponse, error) {
			out, err := packageAddon(args)
			if err != nil {
				return toolErrorResponse("PACKAGE_FAILED", err.Error(), false), nil
			}
			b, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

func packageAddon(args PackageAddonInput) (*PackageAddonOutput, error) {
	projectPath := strings.TrimSpace(args.ProjectPath)
	outputPath := strings.TrimSpace(args.OutputPath)
	if projectPath == "" || outputPath == "" {
		return nil, fmt.Errorf("project_path and output_path are required")
	}

	paths := make([]string, 0)
	for _, dir := range []string{"behavior_pack", "resource_pack"} {
		root := filepath.Join(projectPath, dir)
		if !fileExists(root) {
			continue
		}
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, rerr := filepath.Rel(projectPath, path)
			if rerr != nil {
				return rerr
			}
			paths = append(paths, rel)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if args.DryRun {
		return &PackageAddonOutput{OutputPath: outputPath, Included: paths, DryRun: true}, nil
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return nil, err
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, rel := range paths {
		src := filepath.Join(projectPath, rel)
		in, err := os.Open(src)
		if err != nil {
			return nil, err
		}
		w, err := zw.Create(filepath.ToSlash(rel))
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

	return &PackageAddonOutput{OutputPath: outputPath, Included: paths, DryRun: false}, nil
}

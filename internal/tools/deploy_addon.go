package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"
)

type DeployAddonInput struct {
	ProjectPath string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
	MCDevPath   string `json:"mcdev_path" mcp:"required,description='Path to com.mojang folder'"`
	BPDestName  string `json:"bp_dest_name" mcp:"description='Destination behavior pack folder name'"`
	RPDestName  string `json:"rp_dest_name" mcp:"description='Destination resource pack folder name'"`
	DryRun      bool   `json:"dry_run" mcp:"description='If true, previews deploy without writing'"`
}

type DeployAddonOutput struct {
	Operations []string `json:"operations"`
	DryRun     bool     `json:"dry_run"`
}

func RegisterDeployAddon(server *mcp.Server) error {
	return server.RegisterTool("deploy_addon",
		"Deploys behavior/resource packs to Minecraft development folders.",
		func(args DeployAddonInput) (*mcp.ToolResponse, error) {
			out, err := deployAddon(args)
			if err != nil {
				return toolErrorResponse("DEPLOY_FAILED", err.Error(), false), nil
			}
			b, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

func deployAddon(args DeployAddonInput) (*DeployAddonOutput, error) {
	bpSrc := filepath.Join(args.ProjectPath, "behavior_pack")
	rpSrc := filepath.Join(args.ProjectPath, "resource_pack")
	if !fileExists(bpSrc) {
		return nil, fmt.Errorf("behavior_pack folder not found")
	}
	bpName := strings.TrimSpace(args.BPDestName)
	if bpName == "" {
		bpName = "AddonBP"
	}
	rpName := strings.TrimSpace(args.RPDestName)
	if rpName == "" {
		rpName = "AddonRP"
	}
	bpDst := filepath.Join(args.MCDevPath, "development_behavior_packs", bpName)
	rpDst := filepath.Join(args.MCDevPath, "development_resource_packs", rpName)

	ops := []string{fmt.Sprintf("copy %s -> %s", bpSrc, bpDst)}
	if fileExists(rpSrc) {
		ops = append(ops, fmt.Sprintf("copy %s -> %s", rpSrc, rpDst))
	}

	if args.DryRun {
		return &DeployAddonOutput{Operations: ops, DryRun: true}, nil
	}

	if err := copyDir(bpSrc, bpDst); err != nil {
		return nil, err
	}
	if fileExists(rpSrc) {
		if err := copyDir(rpSrc, rpDst); err != nil {
			return nil, err
		}
	}

	return &DeployAddonOutput{Operations: ops, DryRun: false}, nil
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
			return os.MkdirAll(target, 0755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
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

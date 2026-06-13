// Deploy builds script-api-helper, copies it for MCP clients, and syncs project opencode.json.
//
// Usage: go run ./tools/deploy
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const schema = "https://opencode.ai/config.json"

func main() {
	root, err := findModuleRoot()
	if err != nil {
		fatal(err)
	}

	binaryName := "script-api-helper"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	projectBinary := filepath.Join(root, binaryName)
	relativeCommand := "./" + binaryName

	fmt.Printf("Building %s...\n", binaryName)
	build := exec.Command("go", "build", "-o", projectBinary, "./cmd/script-api-helper")
	build.Dir = root
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		os.Exit(1)
	}

	deployDir, err := localBinDir()
	if err != nil {
		fatal(err)
	}
	deployBinary := filepath.Join(deployDir, binaryName)

	stopRunning(binaryName)
	if err := copyFile(projectBinary, deployBinary); err != nil {
		fatal(fmt.Errorf("deploy to %s: %w", deployBinary, err))
	}
	fmt.Printf("Deployed to %s\n", deployBinary)

	if err := syncOpenCodeConfig(root, relativeCommand); err != nil {
		fatal(err)
	}

	fmt.Println("Done. Restart the Script-API-Helper MCP in your client (OpenCode / Cursor).")
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func localBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func stopRunning(binaryName string) {
	fmt.Printf("Stopping any running %s processes...\n", binaryName)
	if runtime.GOOS == "windows" {
		if err := exec.Command("taskkill", "/IM", binaryName).Run(); err != nil {
			if err := exec.Command("taskkill", "/F", "/IM", binaryName).Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Note: could not stop %s (restart the MCP manually if deploy fails).\n", binaryName)
			}
		}
		return
	}
	_ = exec.Command("pkill", "-x", binaryName).Run()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func syncOpenCodeConfig(root, relativeCommand string) error {
	path := filepath.Join(root, "opencode.json")

	var config map[string]any
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if config == nil {
		config = map[string]any{}
	}

	if _, ok := config["$schema"]; !ok {
		config["$schema"] = schema
	}

	mcp, _ := config["mcp"].(map[string]any)
	if mcp == nil {
		mcp = map[string]any{}
		config["mcp"] = mcp
	}

	mcp["Script-API-Helper"] = map[string]any{
		"type":    "local",
		"command": []string{relativeCommand},
		"enabled": true,
	}

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	if err := os.WriteFile(path, out, 0o644); err != nil {
		return err
	}
	fmt.Printf("Updated %s (Script-API-Helper -> %s)\n", path, relativeCommand)
	return nil
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

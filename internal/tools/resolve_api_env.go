package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

type ResolveAPIEnvInput struct {
	Mode             string `json:"mode" mcp:"description='resolve (default) to recommend modules and versions, list-versions to list npm publish versions for a module'"`
	MinecraftVersion string `json:"minecraft_version" mcp:"description='resolve mode: target Minecraft game version (e.g. 1.21.70, 1.26.13)'"`
	ProjectGoal      string `json:"project_goal" mcp:"description='resolve mode: brief description of what the user wants to build'"`
	ComingFromJava   bool   `json:"coming_from_java" mcp:"description='resolve mode: did the user mention Spigot, Paper, Bukkit, or Java?'"`
	Channel          string `json:"channel" mcp:"description='Version channel: stable, beta, preview, or all (list-versions default: all; resolve default: beta)'"`
	Module           string `json:"module" mcp:"description='list-versions mode: @minecraft module name (e.g. @minecraft/server)'"`
	Limit            int    `json:"limit" mcp:"description='list-versions mode: max versions to return (default 30)'"`
}

type ResolveAPIEnvOutput struct {
	MinecraftVersion string   `json:"minecraft_version"`
	ExactNPMVersion  string   `json:"exact_npm_version"`
	ManifestVersion  string   `json:"manifest_version"`
	AvailableModules []string `json:"available_modules"`
	Guardrails       []string `json:"guardrails"`
	ProjectAdvice    string   `json:"project_advice"`
}

func RegisterResolveAPIEnvironment(server *mcp.Server, npmClient *npm.Client) error {
	return server.RegisterTool("resolve_api_environment",
		"Resolves Bedrock Script API versions from live npm data. Use mode=resolve (default) to recommend modules and guardrails for a Minecraft version and project goal. Use mode=list-versions with module to list available npm publish versions (filter by channel: stable, beta, preview, or all).",
		func(args ResolveAPIEnvInput) (*mcp.ToolResponse, error) {
			return handleResolveAPIEnvironment(args, npmClient)
		})
}

func handleResolveAPIEnvironment(args ResolveAPIEnvInput, npmClient *npm.Client) (*mcp.ToolResponse, error) {
	mode := strings.ToLower(strings.TrimSpace(args.Mode))
	if mode == "" {
		mode = "resolve"
	}

	switch mode {
	case "list-versions", "list_versions":
		return handleListAPIVersions(args, npmClient)
	case "resolve":
		return handleResolveAPIEnv(args, npmClient)
	default:
		return toolErrorResponse("INVALID_INPUT", fmt.Sprintf("unknown mode %q (use resolve or list-versions)", args.Mode), false), nil
	}
}

func handleListAPIVersions(args ResolveAPIEnvInput, npmClient *npm.Client) (*mcp.ToolResponse, error) {
	module := strings.TrimSpace(args.Module)
	if module == "" {
		module = "@minecraft/server"
	}

	vm, err := npmClient.FetchVersionMatrix(module)
	if err != nil {
		return toolErrorResponse("VERSION_LIST_FAILED", err.Error(), true), nil
	}

	limit := args.Limit
	if limit <= 0 {
		limit = 30
	}

	ch := args.Channel
	if ch == "" {
		ch = "all"
	}

	out := make([]string, 0, len(vm.Versions))
	for _, v := range vm.Versions {
		if ch == "all" || matchesVersionChannel(v, ch) {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] > out[j] })
	if len(out) > limit {
		out = out[:limit]
	}

	b, _ := json.MarshalIndent(map[string]any{"module": module, "channel": ch, "versions": out}, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
}

func handleResolveAPIEnv(args ResolveAPIEnvInput, npmClient *npm.Client) (*mcp.ToolResponse, error) {
	if strings.TrimSpace(args.MinecraftVersion) == "" {
		return toolErrorResponse("INVALID_INPUT", "minecraft_version is required in resolve mode", false), nil
	}
	if strings.TrimSpace(args.ProjectGoal) == "" {
		return toolErrorResponse("INVALID_INPUT", "project_goal is required in resolve mode", false), nil
	}

	channel := args.Channel
	if channel != "stable" && channel != "beta" && channel != "preview" {
		channel = "beta"
	}

	vm, err := npmClient.FetchVersionMatrix("@minecraft/server")
	if err != nil {
		return toolErrorResponse("NPM_FETCH_FAILED", fmt.Sprintf("failed to fetch npm data: %v", err), true, "Retry in a moment", "Check network connectivity"), nil
	}

	exact, err := npm.ResolveExactVersionForChannel(vm, args.MinecraftVersion, channel)
	if err != nil {
		return toolErrorResponse("VERSION_RESOLVE_FAILED", fmt.Sprintf("failed to resolve version: %v", err), false, "Use channel stable/beta/preview", "Try minecraft_version=latest", "Use mode=list-versions to see available versions"), nil
	}
	resolved := npm.NormalizeVersion(exact)

	availableModules := []string{"@minecraft/server"}
	candidateModules := []string{
		"@minecraft/server-ui",
		"@minecraft/server-net",
		"@minecraft/server-admin",
		"@minecraft/server-gametest",
	}
	availableModules = append(availableModules, candidateModules...)

	guardrails := []string{}
	if args.ComingFromJava {
		guardrails = append(guardrails, "WARNING: Do not use Bukkit/Spigot/Paper APIs. This is Bedrock JavaScript/TypeScript.")
	}
	guardrails = append(guardrails, "WARNING: mojang-minecraft is deprecated. Use @minecraft/server.")

	advice := fmt.Sprintf("Project '%s' should target @minecraft/server@%s (%s channel). ", args.ProjectGoal, resolved, channel)
	if strings.Contains(strings.ToLower(args.ProjectGoal), "ui") || strings.Contains(strings.ToLower(args.ProjectGoal), "menu") {
		advice += "Consider adding @minecraft/server-ui for forms and dialogs."
	}
	if strings.Contains(strings.ToLower(args.ProjectGoal), "network") || strings.Contains(strings.ToLower(args.ProjectGoal), "http") {
		advice += "Consider adding @minecraft/server-net for network requests."
	}

	output := ResolveAPIEnvOutput{
		MinecraftVersion: args.MinecraftVersion,
		ExactNPMVersion:  exact,
		ManifestVersion:  resolved,
		AvailableModules: availableModules,
		Guardrails:       guardrails,
		ProjectAdvice:    advice,
	}

	jsonOut, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
}

func matchesVersionChannel(version, channel string) bool {
	switch channel {
	case "stable":
		return !strings.Contains(version, "-beta") && !strings.Contains(version, "-preview") && !strings.Contains(version, "-rc")
	case "beta":
		return strings.Contains(version, "-beta") && !strings.Contains(version, "-preview")
	case "preview":
		return strings.Contains(version, "-preview")
	default:
		return true
	}
}

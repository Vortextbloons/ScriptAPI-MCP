package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/npm"
)

// ResolveAPIEnvInput is the input schema for Tool 1
type ResolveAPIEnvInput struct {
	MinecraftVersion string `json:"minecraft_version" mcp:"required,description='Target Minecraft version (e.g. 1.21.60 or latest)'"`
	ProjectGoal      string `json:"project_goal" mcp:"required,description='Brief description of what the user wants to build'"`
	ComingFromJava   bool   `json:"coming_from_java" mcp:"description='Did the user mention Spigot, Paper, Bukkit, or Java?'"`
}

// ResolveAPIEnvOutput is the output schema for Tool 1
type ResolveAPIEnvOutput struct {
	MinecraftVersion string   `json:"minecraft_version"`
	NPMVersion       string   `json:"npm_version"`
	AvailableModules []string `json:"available_modules"`
	Guardrails       []string `json:"guardrails"`
	ProjectAdvice    string   `json:"project_advice"`
}

// RegisterResolveAPIEnvironment registers Tool 1
func RegisterResolveAPIEnvironment(server *mcp.Server, npmClient *npm.Client) error {
	return server.RegisterTool("resolve_api_environment",
		"Fetches the live npm version matrix and establishes project boundaries. Acts as a sanity check before writing code.",
		func(args ResolveAPIEnvInput) (*mcp.ToolResponse, error) {
			return handleResolveAPIEnvironment(args, npmClient)
		})
}

func handleResolveAPIEnvironment(args ResolveAPIEnvInput, npmClient *npm.Client) (*mcp.ToolResponse, error) {
	// Fetch version matrix from npm
	vm, err := npmClient.FetchVersionMatrix("@minecraft/server")
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error fetching npm data: %v", err))), nil
	}

	// Resolve version
	resolved, err := npm.ResolveVersion(vm, args.MinecraftVersion)
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error resolving version: %v", err))), nil
	}

	// Determine available modules for this version
	availableModules := []string{"@minecraft/server"}
	candidateModules := []string{
		"@minecraft/server-ui",
		"@minecraft/server-net",
		"@minecraft/server-admin",
		"@minecraft/server-gametest",
	}
	availableModules = append(availableModules, candidateModules...)

	// Build guardrails
	guardrails := []string{}
	if args.ComingFromJava {
		guardrails = append(guardrails, "WARNING: Do not use Bukkit/Spigot/Paper APIs. This is Bedrock JavaScript/TypeScript.")
	}
	guardrails = append(guardrails, "WARNING: mojang-minecraft is deprecated. Use @minecraft/server.")

	// Project advice
	advice := fmt.Sprintf("Project '%s' should target @minecraft/server@%s. ", args.ProjectGoal, resolved)
	if strings.Contains(strings.ToLower(args.ProjectGoal), "ui") || strings.Contains(strings.ToLower(args.ProjectGoal), "menu") {
		advice += "Consider adding @minecraft/server-ui for forms and dialogs."
	}
	if strings.Contains(strings.ToLower(args.ProjectGoal), "network") || strings.Contains(strings.ToLower(args.ProjectGoal), "http") {
		advice += "Consider adding @minecraft/server-net for network requests."
	}

	output := ResolveAPIEnvOutput{
		MinecraftVersion: args.MinecraftVersion,
		NPMVersion:       resolved,
		AvailableModules: availableModules,
		Guardrails:       guardrails,
		ProjectAdvice:    advice,
	}

	jsonOut, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
}

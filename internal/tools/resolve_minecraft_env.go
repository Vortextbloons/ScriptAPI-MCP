package tools

import (
	"encoding/json"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/mcdev"
	mcp "github.com/metoro-io/mcp-golang"
)

type ResolveMinecraftEnvInput struct {
	MCDevPath string `json:"mcdev_path" mcp:"description='Optional explicit path to validate. When empty, resolves from MINECRAFT_COM_MOJANG env or platform auto-detection.'"`
}

func RegisterResolveMinecraftEnv(server *mcp.Server) error {
	return server.RegisterTool("resolve_minecraft_env",
		"Resolves the Minecraft Bedrock games data folder (com.mojang / minecraftpe) used for deploy targets. Checks explicit mcdev_path, then MINECRAFT_COM_MOJANG/MCDEV_PATH env, then platform auto-detection (Windows unpackaged + Store/UWP, macOS, Linux). Returns path, source, dev_behavior_packs/dev_resource_packs, and probed candidates. Call this before deploy/distribute instead of guessing paths.",
		func(args ResolveMinecraftEnvInput) (*mcp.ToolResponse, error) {
			res, err := mcdev.Resolve(args.MCDevPath)
			if err != nil {
				return toolErrorResponse("MCDEV_NOT_FOUND", err.Error(), false,
					"Set MINECRAFT_COM_MOJANG to the folder containing development_behavior_packs",
					"Pass mcdev_path explicitly for this call",
				), nil
			}
			b, _ := json.MarshalIndent(res, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

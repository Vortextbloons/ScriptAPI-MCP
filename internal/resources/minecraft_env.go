package resources

import (
	"fmt"
	"strings"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/mcdev"
)

// MinecraftEnv returns the bedrock://env/minecraft resource content.
func MinecraftEnv() string {
	var b strings.Builder
	b.WriteString("# Minecraft Bedrock Environment\n\n")
	b.WriteString("Deploy targets live under the **games data folder**:\n")
	b.WriteString("- `development_behavior_packs/`\n- `development_resource_packs/`\n\n")
	b.WriteString("## Resolution order\n\n")
	b.WriteString("1. Explicit `mcdev_path` tool argument\n")
	b.WriteString("2. `MINECRAFT_COM_MOJANG` env var (`MCDEV_PATH` accepted as alias)\n")
	b.WriteString("3. Platform auto-detection:\n")
	b.WriteString("   - Windows unpackaged: `%APPDATA%\\Minecraft Bedrock\\Users\\Shared\\games\\com.mojang`\n")
	b.WriteString("   - Windows Store/UWP: `%LOCALAPPDATA%\\Packages\\Microsoft.*8wekyb3d8bbwe\\LocalState\\games\\com.mojang`\n")
	b.WriteString("   - macOS: `~/Library/Application Support/minecraftpe`\n")
	b.WriteString("   - Linux: `~/.local/share/minecraftpe`, Flatpak `~/.var/app/com.mojang.Minecraft/data/minecraftpe`\n\n")
	res, err := mcdev.Resolve("")
	if err != nil {
		b.WriteString("## This machine\n\n")
		b.WriteString("Not detected: " + err.Error() + "\n")
		return b.String()
	}
	b.WriteString("## This machine\n\n")
	b.WriteString(fmt.Sprintf("- Path: `%s`\n- Source: `%s`\n- Dev BP: `%s`\n- Dev RP: `%s`\n- Exists: `%v`\n- Platform: `%s`\n",
		res.Path, res.Source, res.DevBP, res.DevRP, res.Exists, res.Platform))
	return b.String()
}

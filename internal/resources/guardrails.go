package resources

// StrictRules returns the content of the bedrock://docs/strict_rules resource
func StrictRules() string {
	return `# Bedrock Script API - Strict Rules

## Rule 1: Java API Prohibition
You MUST NOT use any Java-based Minecraft server APIs in Bedrock Script API projects.
Forbidden APIs include but are not limited to:
- Bukkit / Spigot / Paper / BungeeCord APIs
- Java Edition plugin frameworks
- Any code referencing org.bukkit, org.spigotmc, net.minecraft.server

If you are coming from Java development, remember:
- Bedrock Edition uses JavaScript/TypeScript, not Java.
- The runtime is the Bedrock game engine, not the JVM.

## Rule 2: Deprecated Module Prohibition
You MUST NOT use deprecated Mojang modules. These are explicitly forbidden:
- mojang-minecraft (use @minecraft/server)
- mojang-minecraft-ui (use @minecraft/server-ui)
- mojang-minecraft-server-admin (use @minecraft/server-admin)
- mojang-gametest (use @minecraft/server-gametest)

## Rule 3: Allowed Modules Only
Only the following npm modules are permitted in manifest dependencies:
- @minecraft/server
- @minecraft/server-ui
- @minecraft/server-net
- @minecraft/server-admin
- @minecraft/server-gametest

## Syntax Cheat Sheet

### Events
Subscribe to game events using world.afterEvents or system.run:

import { world } from "@minecraft/server";

world.afterEvents.playerPlaceBlock.subscribe((event) => {
    const player = event.player;
    const block = event.block;
    player.sendMessage("You placed a block!");
});

### Forms (UI)
Create in-game menus with server-ui:

import { ActionFormData } from "@minecraft/server-ui";

const form = new ActionFormData()
    .title("My Menu")
    .body("Choose an option")
    .button("Option 1")
    .button("Option 2");

form.show(player).then((response) => {
    if (response.canceled) return;
    console.warn("Selected: " + response.selection);
});

### Commands
Run commands asynchronously from scripts:

player.runCommandAsync("say Hello from Script API!");

### System Ticks
Use system.runInterval for repeating tasks:

import { system } from "@minecraft/server";

system.runInterval(() => {
    console.warn("Tick!");
}, 20); // Every 20 ticks = 1 second
`
}

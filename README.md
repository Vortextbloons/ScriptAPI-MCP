# Bedrock Script API Helper MCP

<p align="center">
  <img src="https://img.shields.io/badge/version-2.1.0-blue?style=flat-square" alt="Version 2.1.0">
  <img src="https://img.shields.io/badge/go-1.26.2-00ADD8?style=flat-square&logo=go" alt="Go 1.26.2">
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="MIT License">
  <img src="https://img.shields.io/badge/MCP-server-7B68EE?style=flat-square" alt="MCP Server">
  <img src="https://img.shields.io/badge/Bedrock-Script%20API-FF6B35?style=flat-square" alt="Bedrock Script API">
</p>

An **MCP (Model Context Protocol) server** for Minecraft Bedrock Script API development. Provides **12 tools** that help AI assistants scaffold addons, validate manifests, search API types, detect breaking changes, generate UUIDs, package deployments, inspect workspaces, resolve the Minecraft install location, and produce boilerplate code — all backed by live npm registry data.

All error responses are **structured JSON** with machine-readable codes, retryability flags, and actionable suggestions.

---

## Tools

| Tool | Description |
|------|-------------|
| `resolve_api_environment` | Resolves npm versions and recommends modules (`mode=resolve`), or lists publish versions (`mode=list-versions`) |
| `search_api` | Queries TypeScript definitions from npm (`source=registry`) or local `node_modules` (`source=local`) |
| `manifest` | Diagnoses, auto-fixes, or syncs dependencies on `manifest.json` (`mode=diagnose`, `fix`, `sync-deps`) |
| `scaffold_addon` | Full project scaffolding with deploy scripts and file output |
| `check_addon` | Validates workspace and compiled output (`mode=check`), runs builds (`mode=build`), scores health, or troubleshoots loading issues |
| `validate_script_api_usage` | Checks JS/TS imports against manifest dependencies and local Script API declarations |
| `distribute_addon` | Packages `.mcaddon` archives and/or deploys to development folders |
| `resolve_minecraft_env` | Resolves the Minecraft games data folder (deploy target) via explicit path, env var, or platform auto-detection |
| `generate_code` | Lists or generates boilerplate patterns (`mode=list` or `generate`) |
| `generate_uuid` | Generates v4 UUIDs with manifest slot presets |
| `diff_script_api_versions` | Diffs the API surface between two module versions |
| `get_mcp_version` | Reports the MCP server name and version |

---

## Quick Start

```bash
# Clone and build (deploys to ~/.local/bin and syncs project opencode.json)
git clone https://github.com/isaac-org/Script-API-Helper-MCP
cd Script-API-Helper-MCP
go run ./tools/deploy
# or on Windows: .\build.ps1

# Run (stdio transport — connect via MCP client)
./script-api-helper.exe
```

`go run ./tools/deploy` builds `script-api-helper.exe` in the project root, copies it to `%USERPROFILE%\.local\bin\` (for global MCP configs), and updates project `opencode.json` so OpenCode uses `./script-api-helper.exe`. Restart the MCP server in your client after building.

The server speaks **stdio MCP transport**. Configure your MCP client (Claude Desktop, OpenCode, VS Code extension, etc.) to launch it as a subprocess.

### OpenCode (project config)

`tools/deploy` writes `opencode.json` automatically. See `opencode.example.json` for the shape.

### VS Code / Claude Desktop config snippet

```json
{
  "mcpServers": {
    "bedrock-script-api": {
      "command": "C:\\path\\to\\script-api-helper.exe"
    }
  }
}
```

---

## Tool Details

### `resolve_api_environment`
Baseline sanity check before writing any code, or version discovery.

| Mode | Purpose |
|------|---------|
| `resolve` (default) | Takes a Minecraft version and channel (`stable`, `beta`, `preview`), resolves the matching npm version, lists available `@minecraft/*` modules, and applies guardrails |
| `list-versions` | Lists npm publish versions for a module with channel filtering (`stable`, `beta`, `preview`, `all`) |

Returns both `exact_npm_version` (for type lookups and API diffing) and `manifest_version` (for manifest dependencies) in resolve mode.

### `search_api`
Searches TypeScript definitions for `@minecraft/*` packages.

| Source | Behavior |
|--------|----------|
| `registry` (default) | Fetches live `.d.ts` from npm; requires `module` plus exact `version` or `minecraft_version` + `channel` |
| `local` | Reads installed packages from `project_path/node_modules/@minecraft/*` (offline, matches installed versions) |

| Mode | Behavior |
|------|----------|
| `types` (default) | Structured symbol extraction |
| `members` | Grep-style substring match across `.d.ts` lines |
| `index` | Lightweight export catalog (`source=local` only) |

Local search supports `modules` (optional filter), `offset` + `limit` pagination (default 50, max 200), `skipped_modules` when a package cannot be read, and `NO_MATCH` when nothing matches.

### `manifest`
Operates on a `manifest.json` string:

| Mode | Purpose |
|------|---------|
| `diagnose` (default) | Validates against 21+ rules (structure, UUIDs, deprecated modules, node_modules mismatch) |
| `fix` | Applies auto-fixes from doctor findings |
| `sync-deps` | Safely adds/removes `@minecraft/*` module dependencies; rejects deprecated modules |

### `scaffold_addon`
Full project scaffold — everything `init_addon_workspace` does, plus optional `package.json` with esbuild build scripts, TypeScript config, deploy script that copies output to the `com.mojang` folder, and writes everything to disk.

**Safety features:** `dry_run` parameter previews files without writing. `overwrite_existing` controls whether existing files are replaced. Path traversal is blocked. Non-Windows deploy scripts include a portability warning.

### Deployment paths

`mcdev_path` is **optional** on `scaffold_addon` and `distribute_addon`. When omitted, the server resolves the games data folder in this order:

1. Explicit `mcdev_path` argument (always wins)
2. `MINECRAFT_COM_MOJANG` env var (`MCDEV_PATH` accepted as alias)
3. Platform auto-detection:
   - Windows unpackaged: `%APPDATA%\Minecraft Bedrock\Users\Shared\games\com.mojang`
   - Windows Store/UWP: `%LOCALAPPDATA%\Packages\Microsoft.*8wekyb3d8bbwe\LocalState\games\com.mojang`
   - macOS: `~/Library/Application Support/minecraftpe`
   - Linux: `~/.local/share/minecraftpe`, Flatpak `~/.var/app/com.mojang.Minecraft/data/minecraftpe`

Call `resolve_minecraft_env` (or read `bedrock://env/minecraft`) instead of guessing paths. Deploy creates `development_behavior_packs/` / `development_resource_packs/` when missing. Generated `deploy.js` reads `DEPLOY_PATH` from `.env`, then `MINECRAFT_COM_MOJANG` / `MCDEV_PATH`, then the same detection chain, so `npm run deploy` works on other machines too.

### Workspace checks
`diagnose_workspace` returns a structured inventory of an addon workspace:

```json
{
  "project_path": "C:/Projects/MyAddon",
  "has_behavior_pack": true,
  "has_resource_pack": true,
  "language": "typescript",
  "entrypoint": "scripts/main.js",
  "source_entrypoint": "src/main.ts",
  "modules": ["@minecraft/server", "@minecraft/server-ui"]
}
```

`check_addon` performs a read-only validation of workspace layout, TypeScript configuration, and the compiled script path declared in the manifest. `build_addon` explicitly runs `npm run build`, returns its diagnostics, and verifies the same output path. `validate_script_api_usage` scans JS/TS sources, checks `@minecraft/*` imports are declared in the behavior pack manifest, validates imported symbols against installed `node_modules` declarations, and flags deprecated symbols.

### `generate_uuid`
Standalone UUID generator with three operating modes:

| Mode | Behavior |
|------|----------|
| `count` | Generate N plain UUIDs (1-50) |
| `slots` | Assign UUIDs to explicit manifest paths: `["header.uuid", "modules[0].uuid"]` |
| `preset` | Predefined slot sets: `bp_basic` (3), `bp_rp_pair` (5), `script_only` (1) |

Output formats: `plain` (list), `assignments` (slot = uuid), `json` (structured).

### `generate_code`
Lists or generates Bedrock Script API boilerplate in JavaScript or TypeScript.

| Mode | Purpose |
|------|---------|
| `list` | Discover snippet types; filter by `category`, `complexity`, `module`, or `query` |
| `generate` (default) | Produce code for a `snippet_type` (events, forms, custom items, advanced patterns) |

TypeScript output uses `import type`, explicit event parameter types, and `: void` return annotations.

### `bedrock_reference`
Static reference lookups (no network required):

| Topic | Purpose |
|-------|---------|
| `color-codes` | Minecraft `§` color and format codes |
| `best-practices` | Curated Script API performance and architecture guidance |

### `diff_script_api_versions`
Diffs the exported TypeScript API surface between two exact npm publishes. Both versions must be exact publish strings (shorthand like `2.8.0-beta` returns concrete candidates). Detects:

- **Breaking**: removed symbols, removed members, signature changes, required parameter additions
- **Non-breaking**: added symbols, deprecated annotations
- **Possible renames**: uses Levensthein similarity + suffix heuristics (`V2`, `New`, `Legacy`)

Output is sorted, filterable by symbol name, and capped by `max_results`.

### `diagnose_workspace`
Diagnoses common reasons Bedrock packs fail to load. Runs workspace validation and returns findings alongside a checklist of recommended checks:
- Valid manifests
- Unique UUIDs
- Script entry file exists
- Compatible module versions
- Packs deployed to correct `com.mojang` directories

### `distribute_addon`
Packages behavior/resource packs into a `.mcaddon` archive, deploys to Minecraft development folders, or both (`action=package`, `deploy`, or `both`).

**Project layout auto-detect:** The tool looks for the behavior pack, resource pack, and compiled scripts directory in this order:

| Source | Default candidates | Override field |
|---|---|---|
| Behavior pack | `behavior_pack/`, `static/bp/`, `src/bp/`, `packs/bp/`, `addon/bp/` | `bp_source` |
| Resource pack | `resource_pack/`, `static/rp/`, `src/rp/`, `packs/rp/`, `addon/rp/` | `rp_source` |
| Compiled scripts | `dist/`, `build/` (merged into the BP at `scripts/` during packaging) | `scripts_source` |

A pack is "found" when its `manifest.json` exists. Each pack is detected independently — a project can have the BP at the root and the RP at `static/rp/`. The resolved layout is reported under `layout` in the response.

**.mcaddon top-level layout:** When the resolved source folder's basename doesn't match a conventional pack name (e.g. `static/bp` is used but the BP is conventionally `behavior_pack`), the tool rewrites the top-level folder in the archive. Defaults come from the (dev-suffix-applied) manifest `header.name` so the .mcaddon has friendly folders like `Tau Gem Upgrades BP/manifest.json` and `Tau Gem Upgrades RP/manifest.json`. Override with `bp_pack_name` / `rp_pack_name`. The compiled scripts source lands at `<bp_pack_name>/scripts/...`. Set `keep_layout=true` to disable the rewrite and preserve the source folder structure verbatim in the archive.

**Manifest suffix (`manifest_suffix`):** Pass `manifest_suffix=true` to ensure `header.name` ends with `-dev`; pass `false` to strip `-dev` for a production build. Omit the field (or pass `null`) to auto-detect from the current `header.name` (a `-dev` suffix implies dev). A staging copy of the workspace is used during the operation and the source manifests are restored on completion — your working tree is never mutated. The response includes a `dev_suffix` block reporting the original/final name and whether it changed for each pack. The `.mcaddon` filename gets a `-dev` suffix in dev mode (e.g. `MyAddon-dev.mcaddon`).

---
### MCP Resources

The server exposes these read-only resources:

| URI | Description |
|-----|-------------|
| `bedrock://docs/strict_rules` | Bedrock Script API guardrails and syntax cheat sheet |
| `bedrock://docs/module_guide` | Module selection and version guidance for addon projects |
| `bedrock://docs/best_practices` | Performance and architecture guidance |
| `bedrock://docs/color_codes` | Minecraft color and format codes |
| `bedrock://env/minecraft` | Resolved Minecraft games data folder and deploy targets for this machine |

---

## Development

```bash
# Build, deploy, and sync OpenCode config
go run ./tools/deploy

# Build & test
go build ./...
go test ./...

# Run all tests including network-dependent (opt-in)
go test -tags=network ./...
```

---

## License

MIT

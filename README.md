# Bedrock Script API Helper MCP

<p align="center">
  <img src="https://img.shields.io/badge/version-1.9.1-blue?style=flat-square" alt="Version 1.9.1">
  <img src="https://img.shields.io/badge/go-1.26.2-00ADD8?style=flat-square&logo=go" alt="Go 1.26.2">
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="MIT License">
  <img src="https://img.shields.io/badge/MCP-server-7B68EE?style=flat-square" alt="MCP Server">
  <img src="https://img.shields.io/badge/Bedrock-Script%20API-FF6B35?style=flat-square" alt="Bedrock Script API">
</p>

An **MCP (Model Context Protocol) server** for Minecraft Bedrock Script API development. Provides **11 tools** that help AI assistants scaffold addons, validate manifests, search API types, detect breaking changes, generate UUIDs, package deployments, inspect workspaces, and produce boilerplate code — all backed by live npm registry data.

All error responses are **structured JSON** with machine-readable codes, retryability flags, and actionable suggestions.

---

## Tools

| Tool | Description |
|------|-------------|
| `resolve_api_environment` | Resolves npm versions and recommends modules (`mode=resolve`), or lists publish versions (`mode=list-versions`) |
| `search_api` | Queries TypeScript definitions from npm (`source=registry`) or local `node_modules` (`source=local`) |
| `manifest` | Diagnoses, auto-fixes, or syncs dependencies on `manifest.json` (`mode=diagnose`, `fix`, `sync-deps`) |
| `scaffold_addon` | Full project scaffolding with deploy scripts and file output |
| `diagnose_workspace` | Validates workspace and diagnoses pack loading issues |
| `distribute_addon` | Packages `.mcaddon` archives and/or deploys to development folders |
| `generate_code` | Lists or generates boilerplate patterns (`mode=list` or `generate`) |
| `generate_uuid` | Generates v4 UUIDs with manifest slot presets |
| `diff_script_api_versions` | Diffs the API surface between two module versions |
| `bedrock_reference` | Static lookups: color codes (`topic=color-codes`) or best practices (`topic=best-practices`) |
| `get_mcp_version` | Reports the MCP server name and version |

---

## Quick Start

```bash
# Clone and build (deploys to ~/.local/bin and syncs project opencode.json)
git clone https://github.com/isaac-org/Script-API-Helper-MCP
cd Script-API-Helper-MCP
go run ./tools/install/main.go
# or on Windows: .\build.ps1

# Run (stdio transport — connect via MCP client)
./script-api-helper.exe
```

`go run ./tools/install/main.go` builds `script-api-helper.exe` in the project root, copies it to `%USERPROFILE%\.local\bin\` (for global MCP configs), and updates project `opencode.json` so OpenCode uses `./script-api-helper.exe`. Restart the MCP server in your client after building.

The server speaks **stdio MCP transport**. Configure your MCP client (Claude Desktop, OpenCode, VS Code extension, etc.) to launch it as a subprocess.

### OpenCode (project config)

`tools/install` writes `opencode.json` automatically. See `opencode.example.json` for the shape.

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

### `inspect_addon_workspace`
Returns a structured inventory of an addon workspace:

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

### `validate_addon_workspace`
Validates the entire workspace against common rules:
- Behavior pack manifest exists and is parseable
- Script module entry point exists on disk
- Source entry point exists (`src/main.js` or `src/main.ts`)
- TypeScript projects have `tsconfig.json`
- Resource pack manifest exists when expected

Returns structured findings with severity, rule ID, and fixability flag.

### `package_addon`
Cross-platform addon packaging. Walks the `behavior_pack/` and `resource_pack/` directories and creates a `.mcaddon` zip archive. Supports `dry_run` to preview included files without writing. Works on any OS (no PowerShell dependency).

### `deploy_addon`
Deploys behavior/resource packs to Minecraft `development_behavior_packs` and `development_resource_packs` folders. Cleans destination folders before copying. Supports dry-run mode (`dry_run=true`).

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

**Dev-suffix handling (`dev_pack`, required for package/both):** Mirrors `npm run bundle` from `scaffold_addon`. Before packaging, the agent must ask whether this is a dev pack: `dev_pack=false` strips `-dev` from manifest `header.name` and the `.mcaddon` filename (production release); `dev_pack=true` ensures `-dev` on both (dev release). Deploy-only (`action=deploy`) copies packs as-is and does not use `dev_pack`. A staging copy is used when manifest names need to change — source files are never modified. The response includes a `dev_suffix` block when packaging.

---
### MCP Resources

The server exposes these read-only resources:

| URI | Description |
|-----|-------------|
| `bedrock://docs/strict_rules` | Bedrock Script API guardrails and syntax cheat sheet |
| `bedrock://docs/module_guide` | Module selection and version guidance for addon projects |

---

## Development

```bash
# Build, deploy, and sync OpenCode config
go run ./tools/install/main.go

# Build & test
go build ./...
go test ./...

# Run all tests including network-dependent (opt-in)
go test -tags=network ./...
```

---

## License

MIT

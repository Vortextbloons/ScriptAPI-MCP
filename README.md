# Bedrock Script API Helper MCP

<p align="center">
  <img src="https://img.shields.io/badge/version-1.6.0-blue?style=flat-square" alt="Version 1.6.0">
  <img src="https://img.shields.io/badge/go-1.26.2-00ADD8?style=flat-square&logo=go" alt="Go 1.26.2">
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="MIT License">
  <img src="https://img.shields.io/badge/MCP-server-7B68EE?style=flat-square" alt="MCP Server">
  <img src="https://img.shields.io/badge/Bedrock-Script%20API-FF6B35?style=flat-square" alt="Bedrock Script API">
</p>

An **MCP (Model Context Protocol) server** for Minecraft Bedrock Script API development. Provides a toolkit of **21 tools** that help AI assistants scaffold addons, validate manifests, search API types, detect breaking changes, generate UUIDs, package deployments, inspect workspaces, and produce boilerplate code — all backed by live npm registry data.

All error responses are **structured JSON** with machine-readable codes, retryability flags, and actionable suggestions.

---

## Tools

| Tool | Description |
|------|-------------|
| `resolve_api_environment` | Fetches the npm version matrix for a Minecraft version and recommends modules |
| `init_addon_workspace` | Generates folder structure, manifests, and starter code for a new addon |
| `search_api_types` | Queries TypeScript definitions from the live npm package |
| `search_api_members` | Searches API members by text match against TypeScript definitions |
| `sync_manifest_dependencies` | Safely adds/removes script modules from an existing manifest |
| `scaffold_addon` | Full project scaffolding with deploy scripts and file output (dry-run and overwrite-safe) |
| `inspect_addon_workspace` | Returns structured inventory of an addon workspace |
| `validate_addon_workspace` | Validates entire workspace: entrypoints, source files, dependencies, config |
| `package_addon` | Cross-platform .mcaddon packaging via zip |
| `deploy_addon` | Deploys packs to Minecraft development folders with dry-run |
| `get_mcp_version` | Reports the MCP server name and version |
| `generate_uuid` | Generates v4 UUIDs with manifest slot presets |
| `generate_bedrock_snippet` | Produces event handler and component boilerplate (JS/TS) |
| `generate_ui_form` | Generates @minecraft/server-ui form boilerplate (action, modal, message) |
| `generate_custom_item` | Generates custom item component registration code |
| `manifest_doctor` | Validates a manifest for 21 common issues |
| `manifest_fixup` | Applies auto-fixes for doctor-detectable problems |
| `diff_script_api_versions` | Diffs the API surface between two module versions |
| `list_api_versions` | Lists available npm publish versions with channel filtering |
| `troubleshoot_pack_not_loading` | Diagnoses common reasons packs fail to load |
| `project_health_score` | Calculates a health score and highlights risky areas |

---

## Quick Start

```bash
# Clone and build
git clone https://github.com/isaac-org/Script-API-Helper-MCP
cd Script-API-Helper-MCP
go build -o script-api-helper.exe ./cmd/script-api-helper/

# Run (stdio transport — connect via MCP client)
./script-api-helper.exe
```

The server speaks **stdio MCP transport**. Configure your MCP client (Claude Desktop, VS Code extension, etc.) to launch it as a subprocess.

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
Baseline sanity check before writing any code. Takes a Minecraft version and channel (`stable`, `beta`, `preview`), resolves the matching npm version, lists available `@minecraft/*` modules, and applies guardrails (Java API prohibition, deprecated module warnings).

Returns both `exact_npm_version` (for type lookups and API diffing) and `manifest_version` (for manifest dependencies).

### `init_addon_workspace`
Generates a complete addon skeleton: behavior pack manifest, optional resource pack manifest, file structure tree, and starter code (`src/main.js` or `src/main.ts`). Dependencies are resolved against live npm data. Uses the server-wide shared npm client for caching.

### `search_api_types`
Fetches the full `.d.ts` bundle for a `@minecraft/*` version and extracts the definition for a specific symbol (class, interface, function, etc.) plus all types it references. Verifies version exists on npm and auto-resolves ambiguous versions.

### `search_api_members`
Searches API members by case-insensitive text match against TypeScript definitions for a module version. Returns matching line snippets with their count. Useful for finding all APIs containing a keyword like `scoreboard`, `location`, or `spawn`.

### `sync_manifest_dependencies`
Parses an existing `manifest.json`, applies add/remove operations on `@minecraft/*` module dependencies, and rejects deprecated modules (`mojang-minecraft`, etc.).

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

### `generate_bedrock_snippet`
Boilerplate code for 6 common patterns, in JavaScript or TypeScript:

| Snippet | Description |
|---------|-------------|
| `beforeEvents.playerBreakBlock` | Block break event handler |
| `afterEvents.playerSpawn` | Player spawn event handler |
| `worldInitialize` | Custom component registration |
| `custom_item_template` | Item component with `onUse` |
| `custom_block_template` | Block component with `onPlayerDestroy` |
| `script_event_handler` | Script event receive handler |

TypeScript output uses `import type`, explicit event parameter types, and `: void` return annotations.

### `generate_ui_form`
Generates `@minecraft/server-ui` form boilerplate. Supports three form types:

| Form Type | Description |
|-----------|-------------|
| `action` | ActionFormData with buttons |
| `modal` | ModalFormData with text fields |
| `message` | MessageFormData with yes/no buttons |

Output in JavaScript or TypeScript.

### `generate_custom_item`
Generates custom item component registration code for `worldInitialize`. Produces a complete `registerCustomComponent` call with `onUse` handler. Supports JavaScript and TypeScript output with typed event parameters.

### `manifest_doctor`
Two-pass diagnostic engine that checks a `manifest.json` against 21 rules:

- **Structural** (raw JSON): invalid JSON, missing `format_version`, wrong header shape, missing UUIDs, duplicate UUIDs, invalid version arrays
- **Semantic** (typed): deprecated modules, unknown `@minecraft/*` modules, missing `@minecraft/server` for script packs, duplicate dependencies, local `node_modules` mismatch

Returns structured findings with `rule` IDs consumable by `manifest_fixup`.

### `manifest_fixup`
Consumes doctor findings and auto-corrects 16 fixable rules — generates missing UUIDs, inserts placeholder names, fixes version arrays, replaces deprecated modules, deduplicates dependencies, and more. Supports selective fix application or bulk `fix_all`.

### `diff_script_api_versions`
Diffs the exported TypeScript API surface between two exact npm publishes. Both versions must be exact publish strings (shorthand like `2.8.0-beta` returns concrete candidates). Detects:

- **Breaking**: removed symbols, removed members, signature changes, required parameter additions
- **Non-breaking**: added symbols, deprecated annotations
- **Possible renames**: uses Levensthein similarity + suffix heuristics (`V2`, `New`, `Legacy`)

Output is sorted, filterable by symbol name, and capped by `max_results`.

### `list_api_versions`
Lists available npm publish versions for a `@minecraft/*` module with channel filtering. Filter by `stable`, `beta`, `preview`, or `all`. Results are sorted highest-first and capped by `limit` (default 30).

### `troubleshoot_pack_not_loading`
Diagnoses common reasons Bedrock packs fail to load. Runs workspace validation and returns findings alongside a checklist of recommended checks:
- Valid manifests
- Unique UUIDs
- Script entry file exists
- Compatible module versions
- Packs deployed to correct `com.mojang` directories

### `project_health_score`
Calculates an addon workspace health score (0-100) from validation findings. Returns a status label (`excellent`, `good`, `fair`, `poor`) and the full findings list. Each error costs 25 points, each warning costs 10.

---

## Architecture

```
cmd/script-api-helper/main.go     ← entry point
└── internal/
    ├── app/app.go                ← lifecycle (signal handling)
    ├── server/server.go          ← MCP tool registration hub
    ├── tools/*.go                ← 21 tool handlers (one file per tool)
    │   ├── resolve_api_env.go
    │   ├── init_addon_workspace.go
    │   ├── search_api_types.go
    │   ├── search_api_members.go
    │   ├── sync_manifest_deps.go
    │   ├── scaffold_addon.go
    │   ├── inspect_addon_workspace.go
    │   ├── validate_addon_workspace.go
    │   ├── package_addon.go
    │   ├── deploy_addon.go
    │   ├── version_info.go
    │   ├── generate_uuid.go
    │   ├── generate_bedrock_snippet.go
    │   ├── generate_ui_form.go
    │   ├── generate_custom_item.go
    │   ├── manifest_doctor.go
    │   ├── manifest_fixup.go
    │   ├── diff_script_api_versions.go
    │   ├── list_api_versions.go
    │   ├── troubleshoot_pack_not_loading.go
    │   ├── project_health_score.go
    │   ├── validation.go
    │   └── errors.go
    ├── models/types.go           ← shared domain types
    ├── manifest/
    │   ├── generator.go          ← manifest creation, UUIDs, starter code
    │   └── validator.go          ← module allowlist/blocklist
    ├── manifestdoctor/           ← doctor + fixer engine
    │   ├── models.go             ← finding/fixup types
    │   ├── rules.go              ← 21 rule checks
    │   ├── doctor.go             ← two-pass diagnostic runner
    │   └── fixer.go              ← auto-fix application
    ├── snippets/                 ← boilerplate generator
    │   ├── templates.go          ← 6 snippet definitions (JS + TS)
    │   └── generator.go          ← template rendering + import building
    ├── apidiff/                  ← breaking-change detection
    │   ├── models.go             ← symbol table types
    │   ├── extract.go            ← .d.ts → symbol table (regex)
    │   └── compare.go            ← diff + rename heuristic
    ├── npm/                      ← npm registry client
    │   ├── client.go             ← HTTP + tarball + caching
    │   ├── parser.go             ← version resolution, .d.ts extraction
    │   ├── validator.go          ← local node_modules validation
    │   ├── version_lookup.go     ← exact version + concrete list helpers
    │   └── cache.go              ← TTL-based in-memory cache
    ├── resources/guardrails.go   ← static strict-rules + module guide resources
    └── version/version.go        ← name + current version
```

### MCP Resources

The server exposes these read-only resources:

| URI | Description |
|-----|-------------|
| `bedrock://docs/strict_rules` | Bedrock Script API guardrails and syntax cheat sheet |
| `bedrock://docs/module_guide` | Module selection and version guidance for addon projects |

---

## Development

```bash
# Build & test
go build ./...
go test ./...

# Build binary
go build -o script-api-helper.exe ./cmd/script-api-helper/

# Run all tests including network-dependent (opt-in)
go test -tags=network ./...
```

---

## License

MIT

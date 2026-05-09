# Bedrock Script API Helper MCP

<p align="center">
  <img src="https://img.shields.io/badge/version-1.6.0-blue?style=flat-square" alt="Version 1.6.0">
  <img src="https://img.shields.io/badge/go-1.26.2-00ADD8?style=flat-square&logo=go" alt="Go 1.26.2">
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="MIT License">
  <img src="https://img.shields.io/badge/MCP-server-7B68EE?style=flat-square" alt="MCP Server">
  <img src="https://img.shields.io/badge/Bedrock-Script%20API-FF6B35?style=flat-square" alt="Bedrock Script API">
</p>

An **MCP (Model Context Protocol) server** for Minecraft Bedrock Script API development. Provides a toolkit of 11 tools that help AI assistants scaffold addons, validate manifests, search API types, detect breaking changes, generate UUIDs, and produce boilerplate code — all backed by live npm registry data.

---

## Tools

| Tool | Description |
|------|-------------|
| `resolve_api_environment` | Fetches the npm version matrix for a Minecraft version and recommends modules |
| `init_addon_workspace` | Generates folder structure, manifests, and starter code for a new addon |
| `search_api_types` | Queries TypeScript definitions from the live npm package |
| `sync_manifest_dependencies` | Safely adds/removes script modules from an existing manifest |
| `scaffold_addon` | Full project scaffolding with deploy scripts and file output |
| `get_mcp_version` | Reports the MCP server name and version |
| `generate_uuid` | Generates v4 UUIDs with manifest slot presets |
| `generate_bedrock_snippet` | Produces event handler and component boilerplate (JS/TS) |
| `manifest_doctor` | Validates a manifest for 21 common issues |
| `manifest_fixup` | Applies auto-fixes for doctor-detectable problems |
| `diff_script_api_versions` | Diffs the API surface between two module versions |

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

### `init_addon_workspace`
Generates a complete addon skeleton: behavior pack manifest, optional resource pack manifest, file structure tree, and starter code (`src/main.js` or `src/main.ts`). Dependencies are resolved against live npm data.

### `search_api_types`
Fetches the full `.d.ts` bundle for a `@minecraft/*` version and extracts the definition for a specific symbol (class, interface, function, etc.) plus all types it references.

### `sync_manifest_dependencies`
Parses an existing `manifest.json`, applies add/remove operations on `@minecraft/*` module dependencies, and rejects deprecated modules (`mojang-minecraft`, etc.).

### `scaffold_addon`
Full project scaffold — everything `init_addon_workspace` does, plus optional `package.json` with esbuild build scripts, TypeScript config, deploy script that copies output to the `com.mojang` folder, and writes everything to disk.

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

---

## Architecture

```
cmd/script-api-helper/main.go     ← entry point
└── internal/
    ├── app/app.go                ← lifecycle (signal handling)
    ├── server/server.go          ← MCP tool registration hub
    ├── tools/*.go                ← 11 tool handlers (one file per tool)
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
    ├── resources/guardrails.go   ← static strict-rules resource
    └── version/version.go        ← name + current version
```

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

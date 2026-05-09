package models

// VersionMatrix holds parsed npm version data for a module
type VersionMatrix struct {
	Module   string            `json:"module"`
	Versions []string          `json:"versions"`
	Tags     map[string]string `json:"tags"`
}

// ResolvedVersion is the output of version resolution
type ResolvedVersion struct {
	MinecraftVersion string   `json:"minecraft_version"`
	NPMVersion       string   `json:"npm_version"`
	AvailableModules []string `json:"available_modules"`
	Guardrails       []string `json:"guardrails"`
}

// Manifest represents the structure of a Bedrock manifest.json
type Manifest struct {
	FormatVersion   int             `json:"format_version"`
	Header          ManifestHeader  `json:"header"`
	Modules         []ManifestModule `json:"modules"`
	Dependencies    []Dependency    `json:"dependencies,omitempty"`
}

type ManifestHeader struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	UUID          string `json:"uuid"`
	Version       []int  `json:"version"`
	MinEngineVersion []int `json:"min_engine_version,omitempty"`
}

type ManifestModule struct {
	Type       string `json:"type"`
	UUID       string `json:"uuid"`
	Version    []int  `json:"version"`
	Language   string `json:"language,omitempty"`
	Entry      string `json:"entry,omitempty"`
}

type Dependency struct {
	ModuleName string `json:"module_name,omitempty"`
	UUID       string `json:"uuid,omitempty"`
	Version    string `json:"version,omitempty"`
}

// AddonWorkspace is the output of init_addon_workspace
type AddonWorkspace struct {
	BehaviorPackManifest string            `json:"behavior_pack_manifest"`
	ResourcePackManifest string            `json:"resource_pack_manifest,omitempty"`
	FileStructure        []string          `json:"file_structure"`
	StarterCode          map[string]string `json:"starter_code"`
}

// AllowedModules is the whitelist for sync_manifest_dependencies
var AllowedModules = []string{
	"@minecraft/server",
	"@minecraft/server-ui",
	"@minecraft/server-net",
	"@minecraft/server-admin",
	"@minecraft/server-gametest",
}

// DeprecatedModules are explicitly forbidden
var DeprecatedModules = []string{
	"mojang-minecraft",
	"mojang-minecraft-ui",
	"mojang-minecraft-server-admin",
	"mojang-gametest",
}

// VersionResolutionResult holds the outcome of attempting to resolve a version string.
type VersionResolutionResult struct {
	Requested  string   `json:"requested"`
	Resolved   string   `json:"resolved"`
	Candidates []string `json:"candidates,omitempty"`
	Error      string   `json:"error,omitempty"`
}

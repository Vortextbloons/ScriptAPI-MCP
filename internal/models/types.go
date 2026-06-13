package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Manifest represents the structure of a Bedrock manifest.json
type Manifest struct {
	FormatVersion int              `json:"format_version"`
	Header        ManifestHeader   `json:"header"`
	Modules       []ManifestModule `json:"modules"`
	Dependencies  []Dependency     `json:"dependencies,omitempty"`
}

type ManifestHeader struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	UUID             string `json:"uuid"`
	Version          []int  `json:"version"`
	MinEngineVersion []int  `json:"min_engine_version,omitempty"`
}

type ManifestModule struct {
	Type     string `json:"type"`
	UUID     string `json:"uuid"`
	Version  []int  `json:"version"`
	Language string `json:"language,omitempty"`
	Entry    string `json:"entry,omitempty"`
}

type Dependency struct {
	ModuleName string `json:"module_name,omitempty"`
	UUID       string `json:"uuid,omitempty"`
	Version    string `json:"version,omitempty"`
}

// UnmarshalJSON accepts Bedrock dependency versions encoded as either strings or arrays.
func (d *Dependency) UnmarshalJSON(data []byte) error {
	var tmp struct {
		ModuleName string          `json:"module_name"`
		UUID       string          `json:"uuid"`
		Version    json.RawMessage `json:"version"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	d.ModuleName = tmp.ModuleName
	d.UUID = tmp.UUID
	d.Version = ""

	if len(tmp.Version) == 0 || string(tmp.Version) == "null" {
		return nil
	}

	if tmp.Version[0] == '"' {
		if err := json.Unmarshal(tmp.Version, &d.Version); err != nil {
			return err
		}
		return nil
	}

	if tmp.Version[0] == '[' {
		var parts []int
		if err := json.Unmarshal(tmp.Version, &parts); err != nil {
			return fmt.Errorf("invalid dependency version array: %w", err)
		}
		vals := make([]string, len(parts))
		for i, part := range parts {
			vals[i] = fmt.Sprint(part)
		}
		d.Version = strings.Join(vals, ".")
		return nil
	}

	return json.Unmarshal(tmp.Version, &d.Version)
}

// AllowedModules is the whitelist for manifest sync-deps mode
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

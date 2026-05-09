package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifest"
)

type GenerateUUIDInput struct {
	Count  int      `json:"count" mcp:"description='Number of UUIDs to generate (1-50, default 1). Ignored if slots or preset provided.'"`
	Slots  []string `json:"slots" mcp:"description='Explicit slot paths like [\"header.uuid\",\"modules[0].uuid\"]'"`
	Preset string   `json:"preset" mcp:"description='Preset name: bp_basic, bp_rp_pair, script_only'"`
	Format string   `json:"format" mcp:"description='Output format: plain, assignments, json (default plain)'"`
}

type GenerateUUIDOutput struct {
	UUIDs          []string          `json:"uuids"`
	Assignments    map[string]string `json:"assignments"`
	PresetUsed     string            `json:"preset_used"`
	CopyPasteBlock string            `json:"copy_paste_block"`
}

func RegisterGenerateUUID(server *mcp.Server) error {
	return server.RegisterTool("generate_uuid",
		"Generates v4 UUIDs for Bedrock manifest.json files. Supports presets (bp_basic, bp_rp_pair, script_only), explicit slot paths, and output formats (plain, assignments, json). Use this whenever a new manifest needs unique UUIDs.",
		func(args GenerateUUIDInput) (*mcp.ToolResponse, error) {
			return handleGenerateUUID(args)
		})
}

func handleGenerateUUID(args GenerateUUIDInput) (*mcp.ToolResponse, error) {
	count := args.Count
	format := args.Format
	if format == "" {
		format = "plain"
	}
	if count <= 0 {
		count = 1
	}
	if count > 50 {
		count = 50
	}

	presets := map[string][]string{
		"bp_basic":    {"header.uuid", "modules[0].uuid", "modules[1].uuid"},
		"bp_rp_pair":  {"bp:header.uuid", "bp:modules[0].uuid", "bp:modules[1].uuid", "rp:header.uuid", "rp:modules[0].uuid"},
		"script_only": {"script_module.uuid"},
	}

	var slots []string
	presetUsed := args.Preset

	if presetUsed != "" {
		var ok bool
		slots, ok = presets[presetUsed]
		if !ok {
			return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Invalid preset %q. Valid presets: bp_basic, bp_rp_pair, script_only", presetUsed))), nil
		}
	} else if len(args.Slots) > 0 {
		slots = args.Slots
		presetUsed = ""
	} else {
		slots = nil
		presetUsed = ""
	}

	var uuids []string
	assignments := make(map[string]string)

	if len(slots) > 0 {
		uuids = make([]string, 0, len(slots))
		for _, slot := range slots {
			uid := manifest.GenerateUUID()
			assignments[slot] = uid
			uuids = append(uuids, uid)
		}
	} else {
		uuids = make([]string, 0, count)
		for i := 0; i < count; i++ {
			uuids = append(uuids, manifest.GenerateUUID())
		}
	}

	copyPasteBlock := ""
	switch format {
	case "assignments":
		if len(assignments) > 0 {
			var sb strings.Builder
			sb.WriteString("# UUID Assignments\n")
			for slot, uid := range assignments {
				sb.WriteString(fmt.Sprintf("%s = %s\n", slot, uid))
			}
			copyPasteBlock = sb.String()
		} else {
			var sb strings.Builder
			sb.WriteString("# Generated UUIDs\n")
			for i, uid := range uuids {
				sb.WriteString(fmt.Sprintf("%d: %s\n", i+1, uid))
			}
			copyPasteBlock = sb.String()
		}
	case "json":
		jsonBytes, _ := json.MarshalIndent(assignments, "", "  ")
		if len(assignments) == 0 {
			jsonBytes, _ = json.MarshalIndent(uuids, "", "  ")
		}
		copyPasteBlock = string(jsonBytes)
	default:
		copyPasteBlock = strings.Join(uuids, "\n")
	}

	output := GenerateUUIDOutput{
		UUIDs:          uuids,
		Assignments:    assignments,
		PresetUsed:     presetUsed,
		CopyPasteBlock: copyPasteBlock,
	}

	jsonOut, _ := json.MarshalIndent(output, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
}

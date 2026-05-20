package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"
)

type LookupColorCodeInput struct {
	Search   string `json:"search" mcp:"description='Filter by color name or code (e.g. dark_red, §4, gold, bold, reset)'"`
	Category string `json:"category" mcp:"description='Filter category: color, format, or all (default all)'"`
}

type ColorCodeEntry struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type LookupColorCodeOutput struct {
	Results       []ColorCodeEntry `json:"results"`
	TotalCount    int              `json:"total_count"`
	FilteredCount int              `json:"filtered_count"`
	ReferenceTable string         `json:"reference_table"`
}

var allColorCodes = []ColorCodeEntry{
	// Color codes
	{Code: "§0", Description: "Black", Category: "color"},
	{Code: "§1", Description: "Dark Blue", Category: "color"},
	{Code: "§2", Description: "Dark Green", Category: "color"},
	{Code: "§3", Description: "Dark Aqua", Category: "color"},
	{Code: "§4", Description: "Dark Red", Category: "color"},
	{Code: "§5", Description: "Dark Purple", Category: "color"},
	{Code: "§6", Description: "Gold", Category: "color"},
	{Code: "§7", Description: "Gray", Category: "color"},
	{Code: "§8", Description: "Dark Gray", Category: "color"},
	{Code: "§9", Description: "Blue", Category: "color"},
	{Code: "§a", Description: "Green", Category: "color"},
	{Code: "§b", Description: "Aqua", Category: "color"},
	{Code: "§c", Description: "Red", Category: "color"},
	{Code: "§d", Description: "Light Purple", Category: "color"},
	{Code: "§e", Description: "Yellow", Category: "color"},
	{Code: "§f", Description: "White", Category: "color"},
	{Code: "§g", Description: "Minecoin Gold", Category: "color"},
	// Format codes
	{Code: "§k", Description: "Obfuscated", Category: "format"},
	{Code: "§l", Description: "Bold", Category: "format"},
	{Code: "§o", Description: "Italic", Category: "format"},
	{Code: "§r", Description: "Reset to default", Category: "format"},
}

func RegisterLookupColorCode(server *mcp.Server) error {
	return server.RegisterTool("lookup_color_code",
		"Returns Minecraft Bedrock color and format codes (§). Use optional search to filter by code or description. Essential for adding colored text to in-game messages, titles, signs, books, and scoreboard displays.",
		func(args LookupColorCodeInput) (*mcp.ToolResponse, error) {
			return handleLookupColorCode(args)
		})
}

func handleLookupColorCode(args LookupColorCodeInput) (*mcp.ToolResponse, error) {
	category := strings.ToLower(args.Category)
	search := strings.ToLower(strings.TrimSpace(args.Search))

	var results []ColorCodeEntry
	for _, entry := range allColorCodes {
		if category == "color" && entry.Category != "color" {
			continue
		}
		if category == "format" && entry.Category != "format" {
			continue
		}
		if search != "" {
			codeClean := strings.ToLower(strings.ReplaceAll(entry.Code, "§", ""))
			desc := strings.ToLower(entry.Description)
			if !strings.Contains(desc, search) && !strings.Contains(codeClean, search) && !strings.Contains(strings.ToLower(entry.Code), search) {
				continue
			}
		}
		results = append(results, entry)
	}

	refTable := buildReferenceTable()

	output := LookupColorCodeOutput{
		Results:        results,
		TotalCount:     len(allColorCodes),
		FilteredCount:  len(results),
		ReferenceTable: refTable,
	}

	jsonOut, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error serializing output: %v", err))), nil
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
}

func buildReferenceTable() string {
	var sb strings.Builder
	sb.WriteString("| Code | Description | Category |\n")
	sb.WriteString("|------|-------------|----------|\n")
	for _, entry := range allColorCodes {
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", entry.Code, entry.Description, entry.Category))
	}
	return sb.String()
}

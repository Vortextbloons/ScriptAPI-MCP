package tools

import (
	mcp "github.com/metoro-io/mcp-golang"
)

type BedrockReferenceInput struct {
	Topic    string `json:"topic" mcp:"description='color-codes for Minecraft § color/format codes, best-practices for Script API guidance (default best-practices)'"`
	Search   string `json:"search" mcp:"description='Optional keyword filter'"`
	Category string `json:"category" mcp:"description='color-codes: color|format|all. best-practices: performance_principles|general|performance_optimization|patterns|all'"`
}

func RegisterBedrockReference(server *mcp.Server) error {
	return server.RegisterTool("bedrock_reference",
		"Static Bedrock Script API reference lookups. Use topic=color-codes for Minecraft § color and format codes (filter by search or category). Use topic=best-practices for curated Script API performance and architecture guidance.",
		func(args BedrockReferenceInput) (*mcp.ToolResponse, error) {
			topic := args.Topic
			if topic == "" {
				topic = "best-practices"
			}

			switch topic {
			case "color-codes", "color_codes", "colors":
				return handleLookupColorCode(LookupColorCodeInput{
					Search:   args.Search,
					Category: args.Category,
				})
			case "best-practices", "best_practices", "practices":
				return handleGetBestPractices(GetBestPracticesInput{
					Search:   args.Search,
					Category: args.Category,
				})
			default:
				return toolErrorResponse("INVALID_INPUT", "unknown topic (use color-codes or best-practices)", false), nil
			}
		})
}

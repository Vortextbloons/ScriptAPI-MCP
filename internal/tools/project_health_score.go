package tools

import (
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"
)

type ProjectHealthScoreInput struct {
	ProjectPath string `json:"project_path" mcp:"required,description='Path to addon workspace root'"`
}

func RegisterProjectHealthScore(server *mcp.Server) error {
	return server.RegisterTool("project_health_score",
		"Calculates an addon workspace health score and highlights risky areas.",
		func(args ProjectHealthScoreInput) (*mcp.ToolResponse, error) {
			val, err := validateAddonWorkspace(args.ProjectPath)
			if err != nil {
				return toolErrorResponse("HEALTH_SCORE_FAILED", err.Error(), false), nil
			}
			score := 100
			for _, f := range val.Findings {
				if f.Severity == "error" {
					score -= 25
				} else {
					score -= 10
				}
			}
			if score < 0 {
				score = 0
			}
			status := "excellent"
			switch {
			case score < 50:
				status = "poor"
			case score < 75:
				status = "fair"
			case score < 90:
				status = "good"
			}
			b, _ := json.MarshalIndent(map[string]any{"score": score, "status": status, "findings": val.Findings}, "", "  ")
			return mcp.NewToolResponse(mcp.NewTextContent(string(b))), nil
		})
}

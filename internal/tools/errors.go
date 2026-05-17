package tools

import (
	"encoding/json"

	mcp "github.com/metoro-io/mcp-golang"
)

type ToolError struct {
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Retryable   bool     `json:"retryable"`
	Suggestions []string `json:"suggestions,omitempty"`
}

type ToolErrorEnvelope struct {
	OK    bool      `json:"ok"`
	Error ToolError `json:"error"`
}

func toolErrorResponse(code, message string, retryable bool, suggestions ...string) *mcp.ToolResponse {
	env := ToolErrorEnvelope{
		OK: false,
		Error: ToolError{
			Code:        code,
			Message:     message,
			Retryable:   retryable,
			Suggestions: suggestions,
		},
	}
	b, _ := json.MarshalIndent(env, "", "  ")
	return mcp.NewToolResponse(mcp.NewTextContent(string(b)))
}

package version

const Current = "2.0.0"
const Name = "Script-API-Helper-MCP"

// DisplayName returns the MCP server name with the current version attached.
func DisplayName() string {
	return Name + " v" + Current
}

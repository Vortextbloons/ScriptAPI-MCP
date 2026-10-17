package version

const Current = "2.1.0"
const Name = "Script-API-Helper-MCP"

// DisplayName returns the MCP server name with the current version attached.
func DisplayName() string {
	return Name + " v" + Current
}

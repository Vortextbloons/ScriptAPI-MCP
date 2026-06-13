package version

const Current = "1.9.4"
const Name = "Script-API-Helper-MCP"

// DisplayName returns the MCP server name with the current version attached.
func DisplayName() string {
	return Name + " v" + Current
}

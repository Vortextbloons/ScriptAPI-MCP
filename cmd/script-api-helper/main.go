package main

import (
	"fmt"
	"os"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}

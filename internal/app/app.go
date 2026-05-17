package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/server"
)

// Run starts the MCP server and blocks until shutdown
func Run() error {
	srv, err := server.New()
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// Handle graceful shutdown signals before serving.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	if err := srv.Serve(); err != nil {
		return fmt.Errorf("server serve error: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Bedrock Script API MCP server started. Waiting for requests...")

	<-sigCh
	fmt.Fprintln(os.Stderr, "Shutting down...")
	return nil
}

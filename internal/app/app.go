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

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()

	fmt.Fprintln(os.Stderr, "Bedrock Script API MCP server started. Waiting for requests...")

	select {
	case serveErr := <-errCh:
		if serveErr != nil {
			return fmt.Errorf("server serve error: %w", serveErr)
		}
		return nil
	case <-sigCh:
		fmt.Fprintln(os.Stderr, "Shutting down...")
		return nil
	}
}

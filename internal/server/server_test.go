package server

import (
	"testing"

	"github.com/metoro-io/mcp-golang/transport/stdio"
)

func TestNewWithTransportRegistersServer(t *testing.T) {
	tr := stdio.NewStdioServerTransport()
	srv, err := NewWithTransport(tr)
	if err != nil {
		t.Fatalf("NewWithTransport returned error: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/metoro-io/mcp-golang/transport/stdio"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/server"
	"github.com/isaac-org/Script-API-Helper-MCP/internal/version"
)

func TestToolCalls(t *testing.T) {
	inReader, inWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create input pipe: %v", err)
	}
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create output pipe: %v", err)
	}

	transport := stdio.NewStdioServerTransportWithIO(inReader, outWriter)
	srv, err := server.NewWithTransport(transport)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	go func() {
		if err := srv.Serve(); err != nil {
			t.Errorf("server serve error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	send := func(msg string) {
		inWriter.WriteString(msg + "\n")
	}

	readResponse := func() string {
		scanner := bufio.NewScanner(outReader)
		if scanner.Scan() {
			return scanner.Text()
		}
		return ""
	}

	// Initialize
	send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`)
	send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	readResponse() // initialize response

	// Test Tool 2: init_addon_workspace
	send(`{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"init_addon_workspace","arguments":{"addon_name":"TestAddon","addon_description":"A test addon","needs_custom_blocks_items_entities":true,"needs_ui_menus":true,"scripting_language":"javascript","server_version":"1.21.60"}}}`)
	resp2 := readResponse()
	fmt.Println("Tool 2 response:", resp2)
	if !strings.Contains(resp2, "TestAddon") {
		t.Error("init_addon_workspace response missing addon name")
	}
	if !strings.Contains(resp2, "@minecraft/server-ui") {
		t.Error("init_addon_workspace response missing server-ui dependency")
	}
	if !strings.Contains(resp2, "resource_pack_manifest") {
		t.Error("init_addon_workspace response missing RP manifest")
	}

	// Test Tool 4: sync_manifest_dependencies with valid module
	send(`{"jsonrpc":"2.0","id":20,"method":"tools/call","params":{"name":"sync_manifest_dependencies","arguments":{"current_manifest_json":"{\"format_version\":2,\"header\":{\"name\":\"Test\",\"description\":\"Desc\",\"uuid\":\"11111111-1111-1111-1111-111111111111\",\"version\":[1,0,0]},\"modules\":[{\"type\":\"data\",\"uuid\":\"22222222-2222-2222-2222-222222222222\",\"version\":[1,0,0]},{\"type\":\"script\",\"uuid\":\"33333333-3333-3333-3333-333333333333\",\"version\":[1,0,0],\"language\":\"javascript\",\"entry\":\"scripts/main.js\"}],\"dependencies\":[{\"module_name\":\"@minecraft/server\",\"version\":\"1.21.60\"}]}","added_modules":["@minecraft/server-net"],"removed_modules":[]}}}`)
	resp4 := readResponse()
	fmt.Println("Tool 4 response:", resp4)
	if !strings.Contains(resp4, "@minecraft/server-net") {
		t.Error("sync_manifest_dependencies response missing added module")
	}

	// Test Tool 4: sync_manifest_dependencies with INVALID module
	send(`{"jsonrpc":"2.0","id":21,"method":"tools/call","params":{"name":"sync_manifest_dependencies","arguments":{"current_manifest_json":"{\"format_version\":2,\"header\":{\"name\":\"Test\",\"description\":\"Desc\",\"uuid\":\"11111111-1111-1111-1111-111111111111\",\"version\":[1,0,0]},\"modules\":[{\"type\":\"data\",\"uuid\":\"22222222-2222-2222-2222-222222222222\",\"version\":[1,0,0]},{\"type\":\"script\",\"uuid\":\"33333333-3333-3333-3333-333333333333\",\"version\":[1,0,0],\"language\":\"javascript\",\"entry\":\"scripts/main.js\"}],\"dependencies\":[{\"module_name\":\"@minecraft/server\",\"version\":\"1.21.60\"}]}","added_modules":["mojang-minecraft"],"removed_modules":[]}}}`)
	resp4b := readResponse()
	fmt.Println("Tool 4 invalid response:", resp4b)
	if !strings.Contains(resp4b, "deprecated") {
		t.Error("sync_manifest_dependencies should reject deprecated module")
	}

	// Test Tool 1: resolve_api_environment
	send(`{"jsonrpc":"2.0","id":30,"method":"tools/call","params":{"name":"resolve_api_environment","arguments":{"minecraft_version":"latest","project_goal":"Build a UI menu addon","coming_from_java":true}}}`)
	resp1 := readResponse()
	fmt.Println("Tool 1 response:", resp1)
	if !strings.Contains(resp1, "WARNING") {
		t.Error("resolve_api_environment should include guardrails")
	}

	// Test version reporting tool
	send(`{"jsonrpc":"2.0","id":40,"method":"tools/call","params":{"name":"get_mcp_version","arguments":{}}}`)
	respVersion := readResponse()
	if !strings.Contains(respVersion, version.Current) {
		t.Error("get_mcp_version should return the current version")
	}

	fmt.Println("All tool call tests passed!")
}

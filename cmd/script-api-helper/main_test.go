package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/server"
	mcpstdio "github.com/isaac-org/Script-API-Helper-MCP/internal/transport"
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

	transport := mcpstdio.NewStdioServerTransportWithIO(inReader, outWriter)
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

	reader := bufio.NewReader(outReader)
	readResponse := func() string {
		b, err := reader.Peek(1)
		if err != nil {
			return ""
		}
		if b[0] == '{' {
			line, err := reader.ReadString('\n')
			if err != nil {
				return ""
			}
			return strings.TrimSpace(line)
		}

		contentLength := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return ""
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			name, value, ok := strings.Cut(line, ":")
			if ok && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
				contentLength, _ = strconv.Atoi(strings.TrimSpace(value))
			}
		}
		if contentLength == 0 {
			return ""
		}
		data := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, data); err != nil {
			return ""
		}
		return string(data)
	}

	// Initialize
	send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`)
	send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	readResponse() // initialize response

	// Test Tool 2: scaffold_addon (replaces init_addon_workspace)
	send(`{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"scaffold_addon","arguments":{"addon_name":"TestAddon","addon_description":"A test addon","needs_custom_blocks_items_entities":true,"needs_ui_menus":true,"scripting_language":"javascript","server_version":"1.21.60","mcdev_path":"C:\\test"}}}`)
	resp2 := readResponse()
	fmt.Println("Tool 2 response:", resp2)
	if !strings.Contains(resp2, "TestAddon") {
		t.Error("scaffold_addon response missing addon name")
	}
	if !strings.Contains(resp2, "@minecraft/server-ui") {
		t.Error("scaffold_addon response missing server-ui dependency")
	}
	if !strings.Contains(resp2, "resource_pack_manifest") {
		t.Error("scaffold_addon response missing RP manifest")
	}

	// Test manifest sync-deps mode
	send(`{"jsonrpc":"2.0","id":20,"method":"tools/call","params":{"name":"manifest","arguments":{"mode":"sync-deps","manifest_json":"{\"format_version\":2,\"header\":{\"name\":\"Test\",\"description\":\"Desc\",\"uuid\":\"11111111-1111-1111-1111-111111111111\",\"version\":[1,0,0]},\"modules\":[{\"type\":\"data\",\"uuid\":\"22222222-2222-2222-2222-222222222222\",\"version\":[1,0,0]},{\"type\":\"script\",\"uuid\":\"33333333-3333-3333-3333-333333333333\",\"version\":[1,0,0],\"language\":\"javascript\",\"entry\":\"scripts/main.js\"}],\"dependencies\":[{\"module_name\":\"@minecraft/server\",\"version\":\"1.21.60\"}]}","added_modules":["@minecraft/server-net"],"removed_modules":[]}}}`)
	resp4 := readResponse()
	fmt.Println("manifest sync-deps response:", resp4)
	if !strings.Contains(resp4, "@minecraft/server-net") {
		t.Error("manifest sync-deps response missing added module")
	}

	// Test manifest sync-deps with INVALID module
	send(`{"jsonrpc":"2.0","id":21,"method":"tools/call","params":{"name":"manifest","arguments":{"mode":"sync-deps","manifest_json":"{\"format_version\":2,\"header\":{\"name\":\"Test\",\"description\":\"Desc\",\"uuid\":\"11111111-1111-1111-1111-111111111111\",\"version\":[1,0,0]},\"modules\":[{\"type\":\"data\",\"uuid\":\"22222222-2222-2222-2222-222222222222\",\"version\":[1,0,0]},{\"type\":\"script\",\"uuid\":\"33333333-3333-3333-3333-333333333333\",\"version\":[1,0,0],\"language\":\"javascript\",\"entry\":\"scripts/main.js\"}],\"dependencies\":[{\"module_name\":\"@minecraft/server\",\"version\":\"1.21.60\"}]}","added_modules":["mojang-minecraft"],"removed_modules":[]}}}`)
	resp4b := readResponse()
	fmt.Println("manifest sync-deps invalid response:", resp4b)
	if !strings.Contains(resp4b, "deprecated") {
		t.Error("manifest sync-deps should reject deprecated module")
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

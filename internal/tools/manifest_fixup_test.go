package tools

import (
	"strings"
	"testing"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifestdoctor"
)

func TestManifestFixupMissingUUID(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := manifestdoctor.RunFixup(m, nil, true, nil)
	if !strings.Contains(result.FixedManifest, "uuid") {
		t.Errorf("expected fixed manifest to contain 'uuid', got: %s", result.FixedManifest)
	}
	if len(result.AppliedFixes) == 0 {
		t.Errorf("expected at least one applied fix")
	}
}

func TestManifestFixupFormatVersion(t *testing.T) {
	m := `{"format_version":3,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := manifestdoctor.RunFixup(m, nil, true, nil)
	if strings.Contains(result.FixedManifest, `"format_version": 3`) {
		t.Errorf("format_version should not be 3 after fix, got: %s", result.FixedManifest)
	}
	if !strings.Contains(result.FixedManifest, `"format_version": 2`) {
		t.Errorf("expected format_version to be 2, got: %s", result.FixedManifest)
	}
}

func TestManifestFixupNoChanges(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]},{"type":"script","uuid":"550e8400-e29b-41d4-a716-446655440002","version":[1,0,0],"language":"javascript","entry":"scripts/main.js"}],"dependencies":[{"module_name":"@minecraft/server","version":"latest"}]}`
	result := manifestdoctor.RunFixup(m, nil, true, nil)
	if len(result.AppliedFixes) != 0 {
		t.Errorf("expected no applied fixes, got: %v", result.AppliedFixes)
	}
}

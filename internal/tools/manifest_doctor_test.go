package tools

import (
	"strings"
	"testing"

	"github.com/isaac-org/Script-API-Helper-MCP/internal/manifestdoctor"
)

func TestManifestDoctorValidManifest(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]},{"type":"script","uuid":"550e8400-e29b-41d4-a716-446655440002","version":[1,0,0],"language":"javascript","entry":"scripts/main.js"}],"dependencies":[{"module_name":"@minecraft/server","version":"latest"}]}`
	result := manifestdoctor.RunDoctor(m, nil)
	if !result.OK {
		t.Errorf("expected OK=true, got errors: %v", result.Errors)
	}
}

func TestManifestDoctorMissingUUID(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := manifestdoctor.RunDoctor(m, nil)
	if !hasManifestDoctorRule(result, "missing_header_uuid") {
		t.Errorf("expected 'missing_header_uuid', got errors: %v", result.Errors)
	}
}

func TestManifestDoctorInvalidJSON(t *testing.T) {
	result := manifestdoctor.RunDoctor("not json", nil)
	if !hasManifestDoctorRule(result, "invalid_json") {
		t.Errorf("expected 'invalid_json', got errors: %v", result.Errors)
	}
}

func TestManifestDoctorDeprecatedModule(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}],"dependencies":[{"module_name":"mojang-minecraft","version":"latest"}]}`
	result := manifestdoctor.RunDoctor(m, nil)
	if !hasManifestDoctorRule(result, "deprecated_module") {
		t.Errorf("expected 'deprecated_module', got errors: %v", result.Errors)
	}
}

func hasManifestDoctorRule(result *manifestdoctor.DoctorOutput, rule string) bool {
	for _, e := range result.Errors {
		if strings.Contains(e.Rule, rule) {
			return true
		}
	}
	return false
}

package manifestdoctor

import (
	"strings"
	"testing"
)

func validManifest() string {
	return `{"format_version":2,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]},{"type":"script","uuid":"550e8400-e29b-41d4-a716-446655440002","version":[1,0,0],"language":"javascript","entry":"scripts/main.js"}],"dependencies":[{"module_name":"@minecraft/server","version":"latest"}]}`
}

func TestDoctorValidManifest(t *testing.T) {
	result := RunDoctor(validManifest(), nil)
	if !result.OK {
		t.Errorf("expected OK=true, got errors: %v", result.Errors)
	}
}

func TestDoctorInvalidJSON(t *testing.T) {
	result := RunDoctor("not json", nil)
	if result.OK {
		t.Fatal("expected OK=false")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected errors")
	}
	if !strings.Contains(result.Errors[0].Rule, "invalid_json") {
		t.Errorf("expected 'invalid_json', got %s", result.Errors[0].Rule)
	}
}

func TestDoctorMissingFormatVersion(t *testing.T) {
	m := `{"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := RunDoctor(m, nil)
	if !hasRule(result, "missing_format_version") {
		t.Errorf("expected 'missing_format_version', got errors: %v", result.Errors)
	}
}

func TestDoctorWrongFormatVersion(t *testing.T) {
	m := `{"format_version":3,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := RunDoctor(m, nil)
	if !hasRule(result, "format_version_not_2") {
		t.Errorf("expected 'format_version_not_2', got errors: %v", result.Errors)
	}
}

func TestDoctorMissingHeader(t *testing.T) {
	m := `{"format_version":2}`
	result := RunDoctor(m, nil)
	if !hasRule(result, "missing_header") {
		t.Errorf("expected 'missing_header', got errors: %v", result.Errors)
	}
}

func TestDoctorMissingHeaderName(t *testing.T) {
	m := `{"format_version":2,"header":{"description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := RunDoctor(m, nil)
	if !hasRule(result, "missing_header_name") {
		t.Errorf("expected 'missing_header_name', got errors: %v", result.Errors)
	}
}

func TestDoctorMissingHeaderUUID(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := RunDoctor(m, nil)
	if !hasRule(result, "missing_header_uuid") {
		t.Errorf("expected 'missing_header_uuid', got errors: %v", result.Errors)
	}
}

func TestDoctorInvalidUUID(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","uuid":"not-a-uuid","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := RunDoctor(m, nil)
	if !hasRule(result, "invalid_header_uuid") {
		t.Errorf("expected 'invalid_header_uuid', got errors: %v", result.Errors)
	}
}

func TestDoctorDuplicateUUID(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0]}]}`
	result := RunDoctor(m, nil)
	if !hasRule(result, "duplicate_uuid") {
		t.Errorf("expected 'duplicate_uuid', got errors: %v", result.Errors)
	}
}

func TestDoctorDeprecatedModule(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}],"dependencies":[{"module_name":"mojang-minecraft","version":"latest"}]}`
	result := RunDoctor(m, nil)
	if !hasRule(result, "deprecated_module") {
		t.Errorf("expected 'deprecated_module', got errors: %v", result.Errors)
	}
}

func TestDoctorMissingMinecraftServer(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"script","uuid":"550e8400-e29b-41d4-a716-446655440002","version":[1,0,0],"language":"javascript","entry":"scripts/main.js"}],"dependencies":[]}`
	result := RunDoctor(m, nil)
	if !hasRule(result, "missing_minecraft_server") {
		t.Errorf("expected 'missing_minecraft_server', got errors: %v", result.Errors)
	}
}

func TestDoctorValidManifestWithUUIDDependencyVersionArray(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"resources","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}],"dependencies":[{"uuid":"550e8400-e29b-41d4-a716-446655440002","version":[2,2,0]}]}`
	result := RunDoctor(m, nil)
	if !result.OK {
		t.Errorf("expected OK=true for UUID dependency version array, got errors: %v", result.Errors)
	}
}

func hasRule(result *DoctorOutput, rule string) bool {
	for _, e := range result.Errors {
		if e.Rule == rule {
			return true
		}
	}
	return false
}

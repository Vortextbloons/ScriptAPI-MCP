package manifestdoctor

import (
	"strings"
	"testing"
)

func TestFixerMissingHeaderUUID(t *testing.T) {
	m := `{"format_version":2,"header":{"name":"Test","description":"Test","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := RunFixup(m, nil, true, nil)
	if len(result.AppliedFixes) == 0 {
		t.Fatal("expected at least one applied fix")
	}
	if !strings.Contains(result.FixedManifest, "uuid") {
		t.Errorf("expected fixed manifest to contain 'uuid', got: %s", result.FixedManifest)
	}
	doc := RunDoctor(result.FixedManifest, nil)
	if hasRule(doc, "missing_header_uuid") {
		t.Errorf("expected no missing_header_uuid after fix, got errors: %v", doc.Errors)
	}
}

func TestFixerFormatVersion(t *testing.T) {
	m := `{"format_version":3,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := RunFixup(m, nil, true, nil)
	if strings.Contains(result.FixedManifest, `"format_version": 3`) {
		t.Errorf("format_version should not be 3 after fix, got: %s", result.FixedManifest)
	}
	if !strings.Contains(result.FixedManifest, `"format_version": 2`) {
		t.Errorf("expected format_version to be 2, got: %s", result.FixedManifest)
	}
	doc := RunDoctor(result.FixedManifest, nil)
	if hasRule(doc, "format_version_not_2") {
		t.Errorf("expected no format_version_not_2 after fix")
	}
}

func TestFixerMissingHeaderName(t *testing.T) {
	m := `{"format_version":2,"header":{"description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := RunFixup(m, nil, true, nil)
	if !strings.Contains(result.FixedManifest, "My Addon") {
		t.Errorf("expected fixed manifest to contain 'My Addon', got: %s", result.FixedManifest)
	}
	doc := RunDoctor(result.FixedManifest, nil)
	if hasRule(doc, "missing_header_name") {
		t.Errorf("expected no missing_header_name after fix")
	}
}

func TestFixerSpecificFixes(t *testing.T) {
	m := `{"format_version":3,"header":{"name":"Test","description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := RunFixup(m, []string{"missing_format_version"}, false, nil)
	if len(result.AppliedFixes) != 1 {
		t.Fatalf("expected 1 applied fix, got %d: %v", len(result.AppliedFixes), result.AppliedFixes)
	}
	if result.AppliedFixes[0].Rule != "missing_format_version" {
		t.Errorf("expected 'missing_format_version', got %s", result.AppliedFixes[0].Rule)
	}
}

func TestFixerNoUnfixableErrors(t *testing.T) {
	m := `{"format_version":2,"header":{"description":"Test","uuid":"550e8400-e29b-41d4-a716-446655440000","version":[1,0,0],"min_engine_version":[1,21,60]},"modules":[{"type":"data","uuid":"550e8400-e29b-41d4-a716-446655440001","version":[1,0,0]}]}`
	result := RunFixup(m, nil, true, nil)
	for _, ue := range result.UnfixableErrors {
		if ue.Rule == "missing_header_name" {
			t.Errorf("expected 'missing_header_name' to not be in unfixable_errors (it is fixable)")
		}
	}
}

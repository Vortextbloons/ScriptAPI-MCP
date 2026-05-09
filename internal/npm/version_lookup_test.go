package npm

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestLookupExactVersionFound(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{Timeout: time.Second},
		cache:      NewCache(),
	}

	vm := &VersionMatrix{
		Module:   "@minecraft/server",
		Versions: []string{"2.6.0", "2.7.0", "2.8.0-beta.1.26.30"},
	}
	data, _ := json.Marshal(vm)
	client.cache.Set("versions:@minecraft/server", data, time.Minute)

	found, err := client.LookupExactVersion("@minecraft/server", "2.7.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected version 2.7.0 to be found")
	}
}

func TestLookupExactVersionNotFound(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{Timeout: time.Second},
		cache:      NewCache(),
	}

	vm := &VersionMatrix{
		Module:   "@minecraft/server",
		Versions: []string{"2.6.0", "2.7.0", "2.8.0-beta.1.26.30"},
	}
	data, _ := json.Marshal(vm)
	client.cache.Set("versions:@minecraft/server", data, time.Minute)

	found, err := client.LookupExactVersion("@minecraft/server", "9.99.99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected version 9.99.99 to not be found")
	}
}

func TestListConcreteVersions(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{Timeout: time.Second},
		cache:      NewCache(),
	}

	vm := &VersionMatrix{
		Module: "@minecraft/server",
		Versions: []string{
			"2.6.0",
			"2.8.0-beta.1.26.20-preview.28",
			"2.9.0-beta.1.26.30-preview.21",
			"2.7.0",
			"2.8.0-beta.1.26.20",
		},
	}
	data, _ := json.Marshal(vm)
	client.cache.Set("versions:@minecraft/server", data, time.Minute)

	results, err := client.ListConcreteVersions("@minecraft/server", "2.8")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(results), results)
	}

	// Should be sorted descending: 2.8.0-beta.1.26.20-preview.28 then 2.8.0-beta.1.26.20
	if results[0] != "2.8.0-beta.1.26.20-preview.28" {
		t.Errorf("expected first result 2.8.0-beta.1.26.20-preview.28, got %q", results[0])
	}
	if results[1] != "2.8.0-beta.1.26.20" {
		t.Errorf("expected second result 2.8.0-beta.1.26.20, got %q", results[1])
	}
}

func TestListConcreteVersionsNoMatch(t *testing.T) {
	client := &Client{
		httpClient: &http.Client{Timeout: time.Second},
		cache:      NewCache(),
	}

	vm := &VersionMatrix{
		Module:   "@minecraft/server",
		Versions: []string{"2.6.0", "2.7.0", "2.8.0-beta.1.26.30"},
	}
	data, _ := json.Marshal(vm)
	client.cache.Set("versions:@minecraft/server", data, time.Minute)

	results, err := client.ListConcreteVersions("@minecraft/server", "9.99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d: %v", len(results), results)
	}
}

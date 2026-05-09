package apidiff

import (
	"fmt"
	"strings"
	"testing"
)

func makeTable(module, version string, symbols []string) *SymbolTable {
	flat := make(map[string]ExportedSymbol)
	var roots []ExportedSymbol
	for _, s := range symbols {
		parts := strings.Split(s, ".")
		sym := ExportedSymbol{Name: s, Kind: KindVariable, Parent: ""}
		flat[s] = sym
		if len(parts) == 1 {
			roots = append(roots, sym)
		}
	}
	return &SymbolTable{Module: module, Version: version, Roots: roots, Flat: flat}
}

func TestCompareTablesEmpty(t *testing.T) {
	from := makeTable("test", "1.0.0", nil)
	to := makeTable("test", "2.0.0", nil)
	result := CompareTables(from, to, false, "", 50)
	if result.Summary != "No changes" {
		t.Errorf("expected 'No changes', got %s", result.Summary)
	}
	if len(result.BreakingChanges) != 0 {
		t.Errorf("expected 0 breaking changes, got %d", len(result.BreakingChanges))
	}
}

func TestCompareTablesSymbolRemoved(t *testing.T) {
	from := makeTable("test", "1.0.0", []string{"foo", "bar"})
	to := makeTable("test", "2.0.0", []string{"foo"})
	result := CompareTables(from, to, false, "", 50)
	if len(result.BreakingChanges) != 1 {
		t.Fatalf("expected 1 breaking change, got %d", len(result.BreakingChanges))
	}
	if result.BreakingChanges[0].Kind != Removed {
		t.Errorf("expected Removed, got %s", result.BreakingChanges[0].Kind)
	}
	if result.BreakingChanges[0].Symbol != "bar" {
		t.Errorf("expected 'bar', got %s", result.BreakingChanges[0].Symbol)
	}
}

func TestCompareTablesSymbolAdded(t *testing.T) {
	from := makeTable("test", "1.0.0", []string{"foo"})
	to := makeTable("test", "2.0.0", []string{"foo", "bar"})
	result := CompareTables(from, to, true, "", 50)
	if len(result.NonBreaking) != 1 {
		t.Fatalf("expected 1 non-breaking change, got %d", len(result.NonBreaking))
	}
	if result.NonBreaking[0].Kind != Added {
		t.Errorf("expected Added, got %s", result.NonBreaking[0].Kind)
	}
	if result.NonBreaking[0].Symbol != "bar" {
		t.Errorf("expected 'bar', got %s", result.NonBreaking[0].Symbol)
	}
}

func TestCompareTablesNoChanges(t *testing.T) {
	from := makeTable("test", "1.0.0", []string{"foo", "bar"})
	to := makeTable("test", "2.0.0", []string{"foo", "bar"})
	result := CompareTables(from, to, true, "", 50)
	if len(result.BreakingChanges) != 0 {
		t.Errorf("expected 0 breaking changes, got %d", len(result.BreakingChanges))
	}
	if len(result.NonBreaking) != 0 {
		t.Errorf("expected 0 non-breaking changes, got %d", len(result.NonBreaking))
	}
}

func TestCompareTablesRenameHeuristic(t *testing.T) {
	from := makeTable("test", "1.0.0", []string{"Player.teleportOld"})
	to := makeTable("test", "2.0.0", []string{"Player.teleport"})
	result := CompareTables(from, to, true, "", 50)
	if len(result.PossibleRenames) == 0 {
		t.Fatalf("expected at least 1 possible rename, got 0")
	}
	if result.PossibleRenames[0].Removed != "Player.teleportOld" {
		t.Errorf("expected removed 'Player.teleportOld', got %s", result.PossibleRenames[0].Removed)
	}
	if result.PossibleRenames[0].Added != "Player.teleport" {
		t.Errorf("expected added 'Player.teleport', got %s", result.PossibleRenames[0].Added)
	}
}

func TestCompareTablesFilter(t *testing.T) {
	from := makeTable("test", "1.0.0", []string{"apple", "banana"})
	to := makeTable("test", "2.0.0", []string{"banana"})
	result := CompareTables(from, to, false, "apple", 50)
	if len(result.BreakingChanges) != 1 {
		t.Fatalf("expected 1 breaking change, got %d", len(result.BreakingChanges))
	}
	if result.BreakingChanges[0].Symbol != "apple" {
		t.Errorf("expected 'apple', got %s", result.BreakingChanges[0].Symbol)
	}
}

func TestCompareTablesMaxResults(t *testing.T) {
	fromSymbols := make([]string, 100)
	for i := 0; i < 100; i++ {
		fromSymbols[i] = fmt.Sprintf("sym%d", i)
	}
	from := makeTable("test", "1.0.0", fromSymbols)
	to := makeTable("test", "2.0.0", nil)
	result := CompareTables(from, to, false, "", 3)
	if len(result.BreakingChanges) > 3 {
		t.Errorf("expected at most 3 breaking changes, got %d", len(result.BreakingChanges))
	}
}

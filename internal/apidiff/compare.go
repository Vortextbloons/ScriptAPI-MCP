package apidiff

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type ChangeKind string

const (
	Removed           ChangeKind = "removed"
	Added             ChangeKind = "added"
	SignatureChanged  ChangeKind = "signature_changed"
	TypeChanged       ChangeKind = "type_changed"
	DeprecatedAdded   ChangeKind = "deprecated_added"
	DeprecatedRemoved ChangeKind = "deprecated_removed"
)

type Change struct {
	Kind          ChangeKind `json:"kind"`
	Symbol        string     `json:"symbol"`
	Details       string     `json:"details"`
	MigrationHint string     `json:"migration_hint,omitempty"`
}

type Rename struct {
	Removed    string `json:"removed"`
	Added      string `json:"added"`
	Confidence string `json:"confidence"`
}

type DiffResult struct {
	Module          string   `json:"module"`
	RequestedFrom   string   `json:"requested_from_version"`
	ResolvedFrom    string   `json:"resolved_from_version"`
	RequestedTo     string   `json:"requested_to_version"`
	ResolvedTo      string   `json:"resolved_to_version"`
	FromVerified    bool     `json:"from_version_verified"`
	ToVerified      bool     `json:"to_version_verified"`
	BreakingChanges []Change `json:"breaking_changes"`
	NonBreaking     []Change `json:"non_breaking_changes"`
	PossibleRenames []Rename `json:"possible_renames"`
	Summary         string   `json:"summary"`
}

func CompareTables(from, to *SymbolTable, includeNonBreaking bool, filter string, maxResults int) *DiffResult {
	if maxResults <= 0 {
		maxResults = 50
	}
	if maxResults > 200 {
		maxResults = 200
	}

	fromNames := from.GetFlatNames()
	toNames := to.GetFlatNames()

	fromSet := make(map[string]bool)
	for _, n := range fromNames {
		fromSet[n] = true
	}
	toSet := make(map[string]bool)
	for _, n := range toNames {
		toSet[n] = true
	}

	var breaking, nonBreaking []Change
	var renames []Rename

	for _, name := range fromNames {
		if !toSet[name] {
			breaking = append(breaking, Change{
				Kind:    Removed,
				Symbol:  name,
				Details: "Symbol removed",
			})
		}
	}

	for _, name := range toNames {
		if !fromSet[name] {
			c := Change{
				Kind:    Added,
				Symbol:  name,
				Details: "Symbol added",
			}
			nonBreaking = append(nonBreaking, c)
		}
	}

	for _, name := range fromNames {
		fromSym, fromOk := from.Flat[name]
		toSym, toOk := to.Flat[name]
		if !fromOk || !toOk {
			continue
		}

		if !fromSym.Deprecated && toSym.Deprecated {
			nonBreaking = append(nonBreaking, Change{
				Kind:    DeprecatedAdded,
				Symbol:  name,
				Details: "@deprecated annotation added",
			})
		} else if fromSym.Deprecated && !toSym.Deprecated {
			nonBreaking = append(nonBreaking, Change{
				Kind:    DeprecatedRemoved,
				Symbol:  name,
				Details: "@deprecated annotation removed",
			})
		}

		if fromSym.Signature != "" && toSym.Signature != "" && fromSym.Signature != toSym.Signature {
			breaking = append(breaking, Change{
				Kind:    SignatureChanged,
				Symbol:  name,
				Details: fmt.Sprintf("Signature changed: %q -> %q", truncateSig(fromSym.Signature), truncateSig(toSym.Signature)),
			})
		}
	}

	parentGroups := make(map[string]map[string]string)
	for _, c := range breaking {
		parent := getParent(c.Symbol)
		if _, ok := parentGroups[parent]; !ok {
			parentGroups[parent] = make(map[string]string)
		}
		parentGroups[parent][c.Symbol] = c.Symbol
	}

	addedByParent := make(map[string]map[string]string)
	for _, c := range nonBreaking {
		if c.Kind == Added {
			parent := getParent(c.Symbol)
			if _, ok := addedByParent[parent]; !ok {
				addedByParent[parent] = make(map[string]string)
			}
			addedByParent[parent][getShortName(c.Symbol)] = c.Symbol
		}
	}

	for parent, removed := range parentGroups {
		added, ok := addedByParent[parent]
		if !ok {
			continue
		}
		for removedFull := range removed {
			removedShort := getShortName(removedFull)
			for addedShort, addedFull := range added {
				sim := levenshteinSimilarity(removedShort, addedShort)
				if sim > 0.6 || hasRenameSuffix(removedShort, addedShort) {
					confidence := "low"
					if sim > 0.8 {
						confidence = "medium"
					}
					if hasRenameSuffix(removedShort, addedShort) || sim > 0.9 {
						confidence = "high"
					}
					renames = append(renames, Rename{
						Removed:    removedFull,
						Added:      addedFull,
						Confidence: confidence,
					})
				}
			}
		}
	}

	if filter != "" {
		filterLower := strings.ToLower(filter)
		breaking = filterChanges(breaking, filterLower)
		nonBreaking = filterChanges(nonBreaking, filterLower)
		renames = filterRenames(renames, filterLower)
	}

	total := len(breaking) + len(nonBreaking) + len(renames)
	if total > maxResults {
		if len(breaking) > maxResults {
			breaking = breaking[:maxResults]
			nonBreaking = nil
			renames = nil
		} else if len(breaking)+len(renames) > maxResults {
			renames = renames[:maxResults-len(breaking)]
			nonBreaking = nil
		} else {
			nonBreaking = nonBreaking[:maxResults-len(breaking)-len(renames)]
		}
	}

	var parts []string
	if len(breaking) > 0 {
		parts = append(parts, fmt.Sprintf("%d breaking", len(breaking)))
	}
	if len(nonBreaking) > 0 {
		parts = append(parts, fmt.Sprintf("%d non-breaking", len(nonBreaking)))
	}
	if len(renames) > 0 {
		parts = append(parts, fmt.Sprintf("%d possible rename(s)", len(renames)))
	}
	summary := "No changes"
	if len(parts) > 0 {
		summary = strings.Join(parts, ", ")
	}

	return &DiffResult{
		Module:          from.Module,
		BreakingChanges: breaking,
		NonBreaking:     nonBreaking,
		PossibleRenames: renames,
		Summary:         summary,
	}
}

func getParent(qualified string) string {
	lastDot := strings.LastIndex(qualified, ".")
	if lastDot < 0 {
		return ""
	}
	return qualified[:lastDot]
}

func getShortName(qualified string) string {
	parts := strings.Split(qualified, ".")
	return parts[len(parts)-1]
}

func truncateSig(sig string) string {
	if len(sig) > 60 {
		return sig[:57] + "..."
	}
	return sig
}

func levenshteinSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	distance := levenshteinDistance(a, b)
	maxLen := math.Max(float64(len(a)), float64(len(b)))
	return 1.0 - float64(distance)/maxLen
}

func levenshteinDistance(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func hasRenameSuffix(removed, added string) bool {
	suffixes := []string{"V2", "New", "Legacy", "V1", "2"}
	for _, s := range suffixes {
		if added == removed+s {
			return true
		}
	}
	return false
}

func filterChanges(changes []Change, filter string) []Change {
	var filtered []Change
	for _, c := range changes {
		if strings.Contains(strings.ToLower(c.Symbol), filter) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func filterRenames(renames []Rename, filter string) []Rename {
	var filtered []Rename
	for _, r := range renames {
		if strings.Contains(strings.ToLower(r.Removed), filter) || strings.Contains(strings.ToLower(r.Added), filter) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func SortDiffResult(result *DiffResult) {
	sort.Slice(result.BreakingChanges, func(i, j int) bool {
		return result.BreakingChanges[i].Symbol < result.BreakingChanges[j].Symbol
	})
	sort.Slice(result.NonBreaking, func(i, j int) bool {
		return result.NonBreaking[i].Symbol < result.NonBreaking[j].Symbol
	})
	sort.Slice(result.PossibleRenames, func(i, j int) bool {
		return result.PossibleRenames[i].Removed < result.PossibleRenames[j].Removed
	})
}

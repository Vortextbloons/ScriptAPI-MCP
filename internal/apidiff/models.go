package apidiff

// SymbolKind represents the kind of a TypeScript symbol.
type SymbolKind string

const (
	KindClass       SymbolKind = "class"
	KindInterface   SymbolKind = "interface"
	KindNamespace   SymbolKind = "namespace"
	KindEnum        SymbolKind = "enum"
	KindFunction    SymbolKind = "function"
	KindType        SymbolKind = "type"
	KindVariable    SymbolKind = "variable"
	KindProperty    SymbolKind = "property"
	KindMethod      SymbolKind = "method"
	KindConstructor SymbolKind = "constructor"
	KindEvent       SymbolKind = "event"
)

// ExportedSymbol represents a single exported TypeScript symbol.
type ExportedSymbol struct {
	Name       string           `json:"name"`
	Kind       SymbolKind       `json:"kind"`
	Signature  string           `json:"signature,omitempty"`
	Deprecated bool             `json:"deprecated,omitempty"`
	Members    []ExportedSymbol `json:"members,omitempty"`
	Parent     string           `json:"parent,omitempty"`
}

// SymbolTable holds all exported symbols for a given module version.
type SymbolTable struct {
	Module  string                    `json:"module"`
	Version string                    `json:"version"`
	Roots   []ExportedSymbol          `json:"roots"`
	Flat    map[string]ExportedSymbol `json:"-"`
}

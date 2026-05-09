package manifestdoctor

// Finding represents a single diagnostic finding from the manifest doctor.
type Finding struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"` // "error" or "warning"
	Message  string `json:"message"`
	Path     string `json:"path"`
	Fixable  bool   `json:"fixable"`
}

// DoctorOutput is the result of running the manifest doctor.
type DoctorOutput struct {
	OK               bool      `json:"ok"`
	DetectedPackKind string   `json:"detected_pack_kind"`
	Errors           []Finding `json:"errors"`
	Warnings         []Finding `json:"warnings"`
	Fixes            []string  `json:"fixes"`
	Summary          string    `json:"summary"`
}

// DoctorOptions are optional parameters for the doctor.
type DoctorOptions struct {
	MinEngineVersion  []int
	CheckLocalModules bool
	ProjectPath       string
}

// FixResult describes a single fix that was applied.
type FixResult struct {
	Rule string `json:"rule"`
	Note string `json:"note"`
}

// FixupOutput is the result of running the manifest fixer.
type FixupOutput struct {
	OriginalManifest string      `json:"original_manifest"`
	FixedManifest    string      `json:"fixed_manifest"`
	AppliedFixes     []FixResult `json:"applied_fixes"`
	UnfixableErrors  []Finding   `json:"unfixable_errors"`
	Summary          string      `json:"summary"`
}

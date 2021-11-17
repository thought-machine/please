package core

// A LintResult represents a result from a linter run.
type LintResult struct {
	// The name of the linter (as configured in .plzconfig, but can be overridden by the linter)
	Linter string `json:"linter"`
	// The file this occurred in
	File string `json:"file"`
	// The line it occurred on (1-indexed)
	Line int `json:"line"`
	// The column it occurred on (if present, 1-indexed)
	Col int `json:"col,omitempty"`
	// The severity (currently a passthrough from the linter)
	Severity string `json:"severity"`
	// A code that the linter attaches to this message. May not be present if
	// it doesn't do that (e.g. gofmt has no such concept)
	Code string `json:"code,omitempty"`
	// Autofix patch, in unified diff format
	Patch string `json:"patch,omitempty"`
	// The message of what's going wrong
	Message string `json:"message"`

	y bool
}

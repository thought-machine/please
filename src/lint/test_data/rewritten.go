package core

// A LintResult represents a result from a linter run.
type LintResult struct {
	Linter   string `json:"linter"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col,omitempty"`
	Severity string `json:"severity"`
	Code     string `json:"code,omitempty"`
	Patch    string `json:"patch,omitempty"`
	Message  string `json:"message"`
}

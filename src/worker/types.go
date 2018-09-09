package worker

// A BuildRequest is the message that's sent to a worker indicating that it should start a build.
type BuildRequest struct {
	// The label of the rule to build, i.e. //src/worker:worker
	Rule string `json:"rule"`
	// Labels applies to this rule.
	Labels []string `json:"labels"`
	// The temporary directory to build the target in.
	TempDir string `json:"temp_dir"`
	// List of source files to compile
	Sources []string `json:"srcs"`
	// Compiler options
	Options []string `json:"opts"`
	// True if this message relates to a test.
	Test bool `json:"test"`
}

// A BuildResponse is sent back from the worker on completion.
type BuildResponse struct {
	// The label of the rule to build, i.e. //src/worker:worker
	// Always corresponds to one that was sent out earlier in a request.
	Rule string `json:"rule"`
	// True if build succeeded
	Success bool `json:"success"`
	// Any messages reported. On failure these should indicate what's gone wrong.
	Messages []string `json:"messages"`
}

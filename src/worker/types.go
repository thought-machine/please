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

// A ParseRequest is a request to provide a parse for a single directory that lacks a BUILD file.
// Providers can infer targets from the files that are present.
type ParseRequest struct {
	// The directory the package is based in.
	Dir string `json:"dir"`
}

type ParseResponse struct {
	// The directory of the original parse request. Must match what was sent in the request.
	Dir string `json:"dir"`
	// True if this provider wants to handle the directory. False if it doesn't consider it valid.
	Handled bool `json:"handled"`
	// The contents of the BUILD file that should be assumed for this directory.
	BuildFile string `json:"build_file"`
}

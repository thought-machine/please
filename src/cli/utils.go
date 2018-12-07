package cli

// PrettyOutput determines from input flags whether we should show 'pretty' output (ie. interactive).
func PrettyOutput(interactiveOutput bool, plainOutput bool, verbosity Verbosity) bool {
	if interactiveOutput && plainOutput {
		log.Fatal("Can't pass both --interactive_output and --plain_output")
	}
	return interactiveOutput || (!plainOutput && StdErrIsATerminal && verbosity < 4)
}

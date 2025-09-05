package core

type BuildEntrypoint struct {
	Entrypoint      []string
	ExecCommandArgs []string
	ExitOnErrorArgs []string
	InteractiveArgs []string
}

type BuildEntrypointOpt func(*BuildEntrypoint)

func WithBuildEntrypointEntrypoint(entrypoint []string) BuildEntrypointOpt {
	return func(be *BuildEntrypoint) {
		be.Entrypoint = entrypoint
	}
}

func WithBuildEntrypointExitOnErrorArgs(args []string) BuildEntrypointOpt {
	return func(be *BuildEntrypoint) {
		be.ExitOnErrorArgs = args
	}
}

func WithBuildEntrypointExecCommandArgs(args []string) BuildEntrypointOpt {
	return func(be *BuildEntrypoint) {
		be.ExecCommandArgs = args
	}
}

func WithBuildEntrypointInteractiveArgs(args []string) BuildEntrypointOpt {
	return func(be *BuildEntrypoint) {
		be.InteractiveArgs = args
	}
}

func NewBuildEntrypoint(opts ...BuildEntrypointOpt) *BuildEntrypoint {
	be := &BuildEntrypoint{
		Entrypoint:      []string{},
		ExecCommandArgs: []string{},
		ExitOnErrorArgs: []string{},
		InteractiveArgs: []string{},
	}

	for _, opt := range opts {
		opt(be)
	}

	// Default to Bash if Entrypoint not set.
	if len(be.Entrypoint) < 1 {
		be.Entrypoint = []string{"bash", "--noprofile", "--norc", "-u", "-o", "pipefail"}
		be.ExecCommandArgs = []string{"-c"}
		be.ExitOnErrorArgs = []string{"-e"}
		be.InteractiveArgs = []string{}
	}

	return be
}

type BuildArgv struct{ Argv []string }
type BuildArgvOpt func(*BuildArgv)

func (be *BuildEntrypoint) WithBuildArgvExitOnError() BuildArgvOpt {
	return func(ba *BuildArgv) {
		ba.Argv = append(ba.Argv, be.ExitOnErrorArgs...)
	}
}

func (be *BuildEntrypoint) WithBuildArgvInteractive() BuildArgvOpt {
	return func(ba *BuildArgv) {
		log.Debugf("pre interactive argv: %#v", ba.Argv)
		ba.Argv = append(ba.Argv, be.InteractiveArgs...)

		log.Debugf("post interactive argv: %#v", ba.Argv)
	}
}

func (be *BuildEntrypoint) WithBuildArgvCommand(command string) BuildArgvOpt {
	return func(ba *BuildArgv) {
		ba.Argv = append(ba.Argv, append(be.ExecCommandArgs, command)...)
	}
}

func (be *BuildEntrypoint) BuildArgv(buildState *BuildState, target *BuildTarget, opts ...BuildArgvOpt) ([]string, error) {
	argv := &BuildArgv{Argv: be.Entrypoint}
	for _, opt := range opts {
		opt(argv)
	}

	newArg0, err := ReplaceSequences(buildState, target, argv.Argv[0])
	if err != nil {
		return nil, err
	}
	argv.Argv[0] = newArg0

	return argv.Argv, nil
}

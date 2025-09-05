package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBuildEntrypoint(t *testing.T) {
	var tests = []struct {
		description string
		opts        []BuildEntrypointOpt
		expected    *BuildEntrypoint
	}{
		{
			"DefaultConfigIsBash",
			nil,
			&BuildEntrypoint{
				Entrypoint:      []string{"bash", "--noprofile", "--norc", "-u", "-o", "pipefail"},
				ExecCommandArgs: []string{"-c"},
				ExitOnErrorArgs: []string{"-e"},
				InteractiveArgs: []string{},
			},
		},
		{
			"NuShell",
			[]BuildEntrypointOpt{
				WithBuildEntrypointEntrypoint([]string{"nu", "--no-config-file", "--no-history"}),
				WithBuildEntrypointInteractiveArgs([]string{"--execute", "$env.config.show_banner = false"}),
				WithBuildEntrypointExecCommandArgs([]string{"--commands"}),
			},
			&BuildEntrypoint{
				Entrypoint:      []string{"nu", "--no-config-file", "--no-history"},
				ExecCommandArgs: []string{"--commands"},
				ExitOnErrorArgs: []string{},
				InteractiveArgs: []string{"--execute", "$env.config.show_banner = false"},
			},
		},
		{
			"Powershell",
			[]BuildEntrypointOpt{
				WithBuildEntrypointEntrypoint([]string{"pwsh", "-NoProfile"}),
				WithBuildEntrypointInteractiveArgs([]string{"-Interactive"}),
				WithBuildEntrypointExecCommandArgs([]string{"-Command"}),
			},
			&BuildEntrypoint{
				Entrypoint:      []string{"pwsh", "-NoProfile"},
				ExecCommandArgs: []string{"-Command"},
				ExitOnErrorArgs: []string{},
				InteractiveArgs: []string{"-Interactive"},
			},
		},
		{
			"Elvish",
			[]BuildEntrypointOpt{
				WithBuildEntrypointEntrypoint([]string{"elvish", "-norc"}),
				WithBuildEntrypointInteractiveArgs([]string{"-i"}),
				WithBuildEntrypointExecCommandArgs([]string{"-c"}),
			},
			&BuildEntrypoint{
				Entrypoint:      []string{"elvish", "-norc"},
				ExecCommandArgs: []string{"-c"},
				ExitOnErrorArgs: []string{},
				InteractiveArgs: []string{"-i"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			actualBec := NewBuildEntrypoint(tt.opts...)
			assert.Equal(t, tt.expected, actualBec)
		})
	}
}

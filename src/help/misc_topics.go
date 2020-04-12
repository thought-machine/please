package help

const plzconfig = `
The root of a Please repo is identified by a ${CYAN}.plzconfig${RESET} file. This also has a number of options to control various ways it behaves.

See ${BLUE}https://please.build/config.html${RESET} for a detailed reference of all options.

There are several different .plzconfig files that can be loaded, which override one another. From lowest to highest priority:
 ${CYAN}.plzconfig${RESET}, which identifies the repo root.
 ${CYAN}.plzconfig_linux_amd64${RESET} (or ${CYAN}.plzconfig_darwin_amd64${RESET}, etc) defines arch-specific options.
 ${CYAN}/etc/please/plzconfig${RESET} can be used to define machine-specific options (e.g. on a CI server)
 ${CYAN}.plzconfig.local${RESET} is used for non-checked-in config that is bespoke to the user.
`

const tracing = `
Please can generate output compatible with Chrome's built-in tracing tool. It can be switched on with the ${BOLD_CYAN}--trace_file${RESET} flag and, once done, you can load the file by visiting ${BLUE}chrome://tracing${RESET}.
This is a handy way to visualise where time is spent during a build and can be useful to diagnose slow builds.
`

const toplevel = `
${BOLD_GREEN}Please${RESET} ${BOLD_WHITE}is a high-performance language-agnostic build system.${RESET}

Try ${BOLD_CYAN}plz help <topic>${RESET} for help on a specific topic;
${BOLD_CYAN}plz --help${RESET} if you want information on flags / options / commands that it accepts;
${BOLD_CYAN}plz help topics${RESET} if you want to see the list of possible topics to get help on
or try a few commands like ${BOLD_CYAN}plz build${RESET} or ${BOLD_CYAN}plz test${RESET} if your repo is already set up and you'd like to see it in action.

Or see the website (${BLUE}https://please.build${RESET}) for more information.
`

var miscTopics = helpSection{
	Topics: map[string]string{
		"plzconfig": plzconfig,
		"tracing": tracing,
		"": toplevel,
	},
}

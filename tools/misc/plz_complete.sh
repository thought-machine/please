####################################################
# plz completion
#
# add
# source <(plz --completion_script)
# to your .bashrc /.zshrc to activate this.
####################################################

_plz_complete_bash() {
    COMP_WORDBREAKS=${COMP_WORDBREAKS//:}
    args=("${COMP_WORDS[@]:1:$COMP_CWORD}")
    local IFS=$'\n'
    COMPREPLY=($(GO_FLAGS_COMPLETION=1 ${COMP_WORDS[0]} "${args[@]}"))
    return 0
}

_plz_complete_zsh() {
    local args=("${words[@]:1:$CURRENT}")
    local IFS=$'\n'
    local completions=($(GO_FLAGS_COMPLETION=1 ${words[1]} "${args[@]}"))
    for completion in $completions; do
	compadd $completion
    done
}

if [ -n "$BASH_VERSION" ]; then
    complete -F _plz_complete_bash plz
elif [ -n "$ZSH_VERSION" ]; then
    compdef _plz_complete_zsh plz
fi

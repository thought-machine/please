#compdef plz
#autoload

####################################################
# plz zsh completion
#
# rename to _plz and put somewhere in your $fpath
# e.g. /usr/local/share/zsh/site-functions
####################################################

local expl
local arguments
local -a _1st_arguments arguments
local options

_1st_arguments=("${(f)$(plz --help \
                        | perl -lnE 'say if (/Available commands/...//)' \
                        | grep '^ ' \
                        | perl -pE 's/^ +//; s/(?<=[^\s])\s+/:/')}")

_targets() {
    completions=("${(f)$(plz query completions --cmd $words[2,-1])}")
    _wanted completions expl "Target" compadd -a completions
}

if [[ $words[-1] =~ '-' ]]; then
    options=("${(f)$(${words[1,-2]} --help \
                      | perl -lnE 'if (/ *(-[a-zA-Z]), ([^ =]*)=? *(.+)/) {say "${1}:$3\n${2}:$3"} elsif (/^ *(--[^ =]*)[ =]*(.*)/) {say "${1}[$2]"}')}")
    _arguments ${options[@]} '1:'
elif (( CURRENT == 2 )); then
  _describe -t commands "plz subcommand" _1st_arguments
  return
elif [[ $words[-2] =~ '^[a-z]' && $words[-2] != 'update' ]]; then
    _targets
fi

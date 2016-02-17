# Bash parameter completion for Please.
#
# Note that colons are fairly crucial to us so some fiddling is needed to keep them
# from counting as separators as it normally would.

_PleaseCompleteMe() {
  local cur
  COMPREPLY=()
  _get_comp_words_by_ref -n : cur

  if [[ "$cur" == -* || "$COMP_CWORD" == "1" || ("$COMP_CWORD" == "2" && "${COMP_WORDS[1]}" == "query")]]; then
      COMPREPLY=( $( compgen -W "`GO_FLAGS_COMPLETION=1 plz ${cur}`" -- $cur) )
  else
      COMPREPLY=( $( compgen -W "`plz --noupdate -p query completions --cmd ${COMP_WORDS[1]} $cur 2>/dev/null`" -- $cur ) )
  fi
  __ltrim_colon_completions "$cur"
  return 0
}

complete -F _PleaseCompleteMe -o filenames plz

# Bash parameter completion for Please.
#
# Note that colons are fairly crucial to us so some fiddling is needed to keep them
# from counting as separators as it normally would.

_PleaseCompleteMe() {
  local cur
  COMPREPLY=()
  _get_comp_words_by_ref -n "/:" cur

  if [[ "$cur" == -* ]]; then
      COMPREPLY=( $( compgen -W "`plz --help 2>&1 | grep -Eo -- '--?[a-z_]+'`" -- $cur ) )
  else
      if [[ "$COMP_CWORD" == "1" ]]; then
	  COMPREPLY=( $( compgen -W "build test cover query clean run update" -- $cur ) )
      else
	  if [[ "$COMP_CWORD" == "2" && "${COMP_WORDS[1]}" == "query" ]]; then
	      COMPREPLY=( $( compgen -W "somepath alltargets deps print completions affectedtests input output" -- $cur ) )
	  else
	      local IFS=$'\n'
	      COMPREPLY=( $( compgen -W "`plz --noupdate -p query completions --cmd ${COMP_WORDS[1]} $cur 2>/dev/null`" -- $cur ) )
	      unset IFS
	  fi
      fi
  fi
  __ltrim_colon_completions "$cur"
  return 0
}

complete -F _PleaseCompleteMe -o filenames plz
COMP_WORDBREAKS=${COMP_WORDBREAKS//:}

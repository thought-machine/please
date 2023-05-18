// Package query implements a set of query operations for Please.
//
// Currently supported operations:
//
//	'deps': 'plz query deps //src:please'
//	        shows the dependency graph of this target.
//	'somepath': 'plz query somepath //src:please //rules:java_rules_pyc'
//	            finds a route between these two targets, if there is one.
//	            useful for saying 'why on earth do I depend on that thing?'
//	'alltargets': 'plz query alltargets //src/...'
//	              shows all targets currently in the graph. careful in large repos!
//	'print': 'plz query print //src:please'
//	         produces a python-like function call that would define the rule.
//	'completions': 'plz query completions //sr'
//	         produces a list of possible completions for the given stem.
//	'changes': 'plz query changes path/to/changed_file.py'
//	         produces a list of targets which have a transitive dependency on
//	         the given file.
//	'input': 'plz query input //src:label' produces a list of all the files
//	         (including transitive deps) that are referenced by this rule.
//	'output': 'plz query output //src:label' produces a list of all the files
//	          that are output by this rule.
//	'graph': 'plz query graph' produces a JSON representation of the build graph
//	         that other programs can interpret for their own uses.
package query

import "github.com/thought-machine/please/src/cli/logging"

var log = logging.Log

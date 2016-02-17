// Basic query language for Please.
//
// Currently supported operations:
//   'deps': 'plz query deps //src:please'
//           shows the dependency graph of this target.
//   'somepath': 'plz query somepath //src:please //src/parse/rules:java_rules_pyc'
//               finds a route between these two targets, if there is one.
//               useful for saying 'why on earth do I depend on that thing?'
//   'alltargets': 'plz query alltargets //src/...'
//                 shows all targets currently in the graph. careful in large repos!
//   'print': 'plz query print //src:please'
//            produces a python-like function call that would define the rule.
//   'completions': 'plz query completions //sr'
//            produces a list of possible completions for the given stem.
//   'affectedtests': 'plz query affectedtests path/to/changed_file.py'
//            produces a list of test targets which have a transitive dependency on
//            the given file.
//   'input': 'plz query input //src:label' produces a list of all the files
//            (including transitive deps) that are referenced by this rule.
package query

import "gopkg.in/op/go-logging.v1"

var log = logging.MustGetLogger("query")

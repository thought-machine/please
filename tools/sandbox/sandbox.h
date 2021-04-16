#include <stdbool.h>

// contain separates the process into new namespaces to sandbox it.
// It should be passed the argv for the new process, and booleans indicating
// whether it should move to new network and mount namespaces.
// It returns an exit code (so 0 on success, nonzero on failure).
int contain(char* argv[], bool net, bool mount);

// exec_name returns the name of the new binary to exec() as.
// old_name is the current name; if it's within old_dir it will be re-prefixed to new_dir.
char* exec_name(const char* old_name, const char* old_dir, const char* new_dir);

// change_path takes a string or environment variable and changes a prefix from one path to another.
// For example:
//   old_name:   RESULTS_FILE=/home/peter/git/please/plz-out/tmp/my_test/test.results
//   old_dir:    /home/peter/git/please/plz-out/tmp/my_test
//   new_dir:    /tmp/plz_sandbox
//   prefix_len: 13
// Result:       RESULTS_FILE=/tmp/plz_sandbox/test.results
char* change_path(const char* old_name, const char* old_dir, const char* new_dir, int prefix_len);

// change_env_vars changes any environment variables prefixed with the old directory to the new one.
// The variables are changed in-place within the given array.
void change_env_vars(char** environ, const char* old_dir, const char* new_dir);

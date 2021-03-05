#include <stdbool.h>

// contain separates the process into new namespaces to sandbox it.
// It should be passed the argv for the new process, and booleans indicating
// whether it should move to new network and mount namespaces.
// It returns an exit code (so 0 on success, nonzero on failure).
int contain(char* argv[], bool net, bool mount);

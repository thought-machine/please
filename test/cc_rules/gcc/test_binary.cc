// Just a wee binary to test that we can compile one ok.

#include <string>

#include "test/cc_rules/gcc/embedded_files.h"


int main(int argc, char** argv) {
    using namespace plz;

    if (embedded_file1_contents() != "testing message 1\n") {
	return 1;
    } else if (embedded_file3_contents() != "testing message 3\n") {
	return 3;
    } else {
	return 0;
    }
}

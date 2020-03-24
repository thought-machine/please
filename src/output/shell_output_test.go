package output

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColouriseError(t *testing.T) {
	err := fmt.Errorf("/opt/tm/toolchains/1.8.2/usr/include/fst/label-reachable.h:176:39: error: non-const lvalue reference to type 'unordered_map<int, int>' cannot bind to a value of unrelated type 'unordered_map<unsigned char, unsigned char>'")
	expected := fmt.Errorf("${BOLD_WHITE}/opt/tm/toolchains/1.8.2/usr/include/fst/label-reachable.h, line 176, column 39:${RESET} ${BOLD_RED}error: ${RESET}${BOLD_WHITE}non-const lvalue reference to type 'unordered_map<int, int>' cannot bind to a value of unrelated type 'unordered_map<unsigned char, unsigned char>'${RESET}")
	assert.EqualValues(t, expected, colouriseError(err))
}

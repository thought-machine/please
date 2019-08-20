package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitedPrintfSimple(t *testing.T) {
	assert.Equal(t, "wibble wobble", lprintfPrepare(20, "wibble wobble"))
}

func TestLimitedPrintfMore(t *testing.T) {
	assert.Equal(t, "wibble ...", lprintfPrepare(10, "wibble wobble"))
}

func TestLimitedPrintfAnsi(t *testing.T) {
	// Should be unchanged because without escape sequences it's under the limit.
	assert.Equal(t, "\x1b[30mwibble wobble\x1b[1m", lprintfPrepare(20, "\x1b[30mwibble wobble\x1b[1m"))
}

func TestLimitedPrintfAnsiNotCountedWhenReducing(t *testing.T) {
	assert.Equal(t, "\x1b[30mwibble ...\x1b[1m", lprintfPrepare(10, "\x1b[30mwibble wobble\x1b[1m"))
}

func TestNewlinesStillWritte(t *testing.T) {
	// Test that newline still gets written (it doesn't count as horizontal space and is Very Important)
	assert.Equal(t, "\x1b[30mwibble ...\x1b[1m\n", lprintfPrepare(10, "\x1b[30mwibble wobble\x1b[1m\n"))
}

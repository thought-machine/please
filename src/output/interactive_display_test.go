package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLimitedPrintfSimple(t *testing.T) {
	var d interactiveDisplay
	assert.Equal(t, "wibble wobble", d.lprintfPrepare(20, "wibble wobble"))
}

func TestLimitedPrintfMore(t *testing.T) {
	var d interactiveDisplay
	assert.Equal(t, "wibble ...", d.lprintfPrepare(10, "wibble wobble"))
}

func TestLimitedPrintfAnsi(t *testing.T) {
	var d interactiveDisplay
	// Should be unchanged because without escape sequences it's under the limit.
	assert.Equal(t, "\x1b[30mwibble wobble\x1b[1m", d.lprintfPrepare(20, "\x1b[30mwibble wobble\x1b[1m"))
}

func TestLimitedPrintfAnsiNotCountedWhenReducing(t *testing.T) {
	var d interactiveDisplay
	assert.Equal(t, "\x1b[30mwibble ...\x1b[1m", d.lprintfPrepare(10, "\x1b[30mwibble wobble\x1b[1m"))
}

func TestNewlinesStillWritte(t *testing.T) {
	var d interactiveDisplay
	// Test that newline still gets written (it doesn't count as horizontal space and is Very Important)
	assert.Equal(t, "\x1b[30mwibble ...\x1b[1m\n", d.lprintfPrepare(10, "\x1b[30mwibble wobble\x1b[1m\n"))
}

func TestLimitedPrintfEraseAfter(t *testing.T) {
	var d interactiveDisplay
	assert.Equal(t, "\x1b[Kmmmmmmm...\n", d.lprintfPrepare(10, "\x1b[Kmmmmmmmmmmmmmmmmmmmm\n"))
}

func TestLimitedPrintfTabsExpanded(t *testing.T) {
	var d interactiveDisplay
	assert.Equal(t, "a       b", d.lprintfPrepare(20, "a\tb"))
	assert.Equal(t, "a       b...", d.lprintfPrepare(12, "a\tbcdefghijk"))
}

func TestLimitedPrintfNewlineResetsWidth(t *testing.T) {
	var d interactiveDisplay
	assert.Equal(t, "wibble ...\nwibble ...\n", d.lprintfPrepare(10, "wibble wobble\nwibble wobble\n"))
}

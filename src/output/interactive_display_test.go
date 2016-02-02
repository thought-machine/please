package output

import "strings"
import "testing"

import "core"

func TestLimitedPrintfSimple(t *testing.T) {
	assertEquals(t, "wibble wobble", lprintfPrepare(20, "wibble wobble"))
	assertEquals(t, "wibble wobble", lprintfPrepare(20, "%s %s", "wibble", "wobble"))
}

func TestLimitedPrintfMore(t *testing.T) {
	assertEquals(t, "wibble ...", lprintfPrepare(10, "wibble wobble"))
	assertEquals(t, "wibble ...", lprintfPrepare(10, "%s %s", "wibble", "wobble"))
}

func TestLimitedPrintfAnsi(t *testing.T) {
	// Should be unchanged because without escape sequences it's under the limit.
	assertEquals(t, "\x1b[30mwibble wobble\x1b[1m", lprintfPrepare(20, "\x1b[30mwibble wobble\x1b[1m"))
}

func TestLimitedPrintfAnsiNotCountedWhenReducing(t *testing.T) {
	assertEquals(t, "\x1b[30mwibble ...\x1b[1m", lprintfPrepare(10, "%s %s", "\x1b[30mwibble", "wobble\x1b[1m"))
}

func TestNewlinesStillWritte(t *testing.T) {
	// Test that newline still gets written (it doesn't count as horizontal space and is Very Important)
	assertEquals(t, "\x1b[30mwibble ...\x1b[1m\n", lprintfPrepare(10, "%s %s\n", "\x1b[30mwibble", "wobble\x1b[1m"))
}

func TestLineWrap(t *testing.T) {
	backend := logBackend{Cols: 80, maxLines: 2}

	s := backend.lineWrap(strings.Repeat("a", 40))
	assertEquals(t, strings.Repeat("a", 40), strings.Join(s, "\n"))

	s = backend.lineWrap(strings.Repeat("a", 100))
	assertEquals(t, strings.Repeat("a", 20)+"\n"+strings.Repeat("a", 80), strings.Join(s, "\n"))

	s = backend.lineWrap(strings.Repeat("a", 80))
	assertEquals(t, strings.Repeat("a", 80), strings.Join(s, "\n"))
}

func assertEquals(t *testing.T, expected, actual string) {
	if expected != actual {
		t.Errorf("Assertion failed: expected %s, was %s", expected, actual)
	}
}

func TestDisplayLabel(t *testing.T) {
	label := core.BuildLabel{PackageName: "src/display", Name: "display"}
	assertEquals(t, "//src/display:display", displayLabel(label))
	label = core.BuildLabel{PackageName: "src/display", Name: "_display#lib"}
	assertEquals(t, "//src/display:display", displayLabel(label))
	label = core.BuildLabel{PackageName: "src/display", Name: "display#lib"}
	assertEquals(t, "//src/display:display#lib", displayLabel(label))
	label = core.BuildLabel{PackageName: "src/display", Name: "_display_lib"}
	assertEquals(t, "//src/display:_display_lib", displayLabel(label))
	label = core.BuildLabel{PackageName: "src/display", Name: ""}
	assertEquals(t, "//src/display", displayLabel(label))
	// Happens sometimes when you start stacking up labels
	label = core.BuildLabel{PackageName: "src/display", Name: "__display#lib#zip"}
	assertEquals(t, "//src/display:display", displayLabel(label))
}

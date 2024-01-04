package asp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertNextToken(t *testing.T, l *lex, tokenType rune, value string, line, column, offset int) {
	t.Helper()
	assertToken(t, l, l.Next(), tokenType, value, line, column, offset)
}

func assertToken(t *testing.T, l *lex, tok Token, tokenType rune, value string, line, column, offset int) {
	t.Helper()
	pos := l.File().Pos(tok.Pos)
	assert.EqualValues(t, tokenType, tok.Type, "incorrect type")
	assert.Equal(t, value, tok.Value, "incorrect value")
	assert.Equal(t, line, pos.Line, "incorrect line")
	assert.Equal(t, column, pos.Column, "incorrect column")
	assert.Equal(t, offset, pos.Offset, "incorrect offset")
}

func TestLexBasic(t *testing.T) {
	l := newLexer(strings.NewReader("hello world"))
	assertNextToken(t, l, Ident, "hello", 1, 1, 1)
	assertToken(t, l, l.Peek(), Ident, "world", 1, 7, 7)
	assertNextToken(t, l, Ident, "world", 1, 7, 7)
	assertNextToken(t, l, EOL, "", 1, 12, 12)
	assertNextToken(t, l, EOF, "", 2, 1, 13)
}

func TestLexMultiline(t *testing.T) {
	l := newLexer(strings.NewReader("hello\nworld\n"))
	assertNextToken(t, l, Ident, "hello", 1, 1, 1)
	assertNextToken(t, l, EOL, "", 1, 6, 6)
	assertNextToken(t, l, Ident, "world", 2, 1, 7)
	assertNextToken(t, l, EOL, "", 2, 6, 12)
	assertNextToken(t, l, EOF, "", 3, 1, 13)
}

const testFunction = `
def func(x):
    pass
`

func TestLexFunction(t *testing.T) {
	l := newLexer(strings.NewReader(testFunction))
	assertNextToken(t, l, Ident, "def", 2, 1, 2)
	assertNextToken(t, l, Ident, "func", 2, 5, 6)
	assertNextToken(t, l, '(', "(", 2, 9, 10)
	assertNextToken(t, l, Ident, "x", 2, 10, 11)
	assertNextToken(t, l, ')', ")", 2, 11, 12)
	assertNextToken(t, l, ':', ":", 2, 12, 13)
	assertNextToken(t, l, EOL, "", 2, 13, 14)
	assertNextToken(t, l, Ident, "pass", 3, 5, 19)
	assertNextToken(t, l, EOL, "", 4, 1, 24)
	assertNextToken(t, l, Unindent, "", 4, 1, 24)
	assertNextToken(t, l, EOF, "", 4, 1, 24)
}

func TestLexUnicode(t *testing.T) {
	l := newLexer(strings.NewReader("懂了吗 你愁脸 有没有"))
	assertNextToken(t, l, Ident, "懂了吗", 1, 1, 1)
	assertNextToken(t, l, Ident, "你愁脸", 1, 11, 11)
	assertNextToken(t, l, Ident, "有没有", 1, 21, 21)
	assertNextToken(t, l, EOL, "", 1, 30, 30)
	assertNextToken(t, l, EOF, "", 2, 1, 31)
}

func TestLexString(t *testing.T) {
	l := newLexer(strings.NewReader(`x = "hello world"`))
	assertNextToken(t, l, Ident, "x", 1, 1, 1)
	assertNextToken(t, l, '=', "=", 1, 3, 3)
	assertNextToken(t, l, String, "\"hello world\"", 1, 5, 5)
	assertNextToken(t, l, EOL, "", 1, 18, 18)
	assertNextToken(t, l, EOF, "", 2, 1, 19)
}

func TestLexStringEscape(t *testing.T) {
	l := newLexer(strings.NewReader(`x = '\n\\'`))
	assertNextToken(t, l, Ident, "x", 1, 1, 1)
	assertNextToken(t, l, '=', "=", 1, 3, 3)
	assertNextToken(t, l, String, "\"\n\\\"", 1, 5, 5)
	assertNextToken(t, l, EOL, "", 1, 11, 11)
	assertNextToken(t, l, EOF, "", 2, 1, 12)
}

func TestLexStringEscape2(t *testing.T) {
	l := newLexer(strings.NewReader(`'echo -n "import \( \";'`))
	assertNextToken(t, l, String, `"echo -n "import \( ";"`, 1, 1, 1)
	assertNextToken(t, l, EOL, "", 1, 25, 25)
	assertNextToken(t, l, EOF, "", 2, 1, 26)
}

func TestLexRawString(t *testing.T) {
	l := newLexer(strings.NewReader(`x = r'\n\\'`))
	assertNextToken(t, l, Ident, "x", 1, 1, 1)
	assertNextToken(t, l, '=', "=", 1, 3, 3)
	assertNextToken(t, l, String, `"\n\\"`, 1, 5, 5)
	assertNextToken(t, l, EOL, "", 1, 12, 12)
	assertNextToken(t, l, EOF, "", 2, 1, 13)
}

func TestLexFString(t *testing.T) {
	l := newLexer(strings.NewReader(`x = f'{x}'`))
	assertNextToken(t, l, Ident, "x", 1, 1, 1)
	assertNextToken(t, l, '=', "=", 1, 3, 3)
	assertNextToken(t, l, String, `f"{x}"`, 1, 5, 5)
	assertNextToken(t, l, EOL, "", 1, 11, 11)
	assertNextToken(t, l, EOF, "", 2, 1, 12)
}

const testMultilineString = `x = """
hello\n
world
"""`

// expected output after lexing; note quotes are broken to a single one and \n does not become a newline.
const expectedMultilineString = `"
hello

world
"`

func TestLexMultilineString(t *testing.T) {
	l := newLexer(strings.NewReader(testMultilineString))
	assertNextToken(t, l, Ident, "x", 1, 1, 1)
	assertNextToken(t, l, '=', "=", 1, 3, 3)
	assertNextToken(t, l, String, expectedMultilineString, 1, 5, 5)
	assertNextToken(t, l, EOL, "", 4, 4, 26)
	assertNextToken(t, l, EOF, "", 5, 1, 27)
}

func TestLexAttributeAccess(t *testing.T) {
	l := newLexer(strings.NewReader(`x.call(y)`))
	assertNextToken(t, l, Ident, "x", 1, 1, 1)
	assertNextToken(t, l, '.', ".", 1, 2, 2)
	assertNextToken(t, l, Ident, "call", 1, 3, 3)
	assertNextToken(t, l, '(', "(", 1, 7, 7)
	assertNextToken(t, l, Ident, "y", 1, 8, 8)
	assertNextToken(t, l, ')', ")", 1, 9, 9)
	assertNextToken(t, l, EOL, "", 1, 10, 10)
	assertNextToken(t, l, EOF, "", 2, 1, 11)
}

func TestLexFunctionArgs(t *testing.T) {
	l := newLexer(strings.NewReader(`def test(name='name', timeout=10, args=CONFIG.ARGS):`))
	assertNextToken(t, l, Ident, "def", 1, 1, 1)
	assertNextToken(t, l, Ident, "test", 1, 5, 5)
	assertNextToken(t, l, '(', "(", 1, 9, 9)
	assertNextToken(t, l, Ident, "name", 1, 10, 10)
	assertNextToken(t, l, '=', "=", 1, 14, 14)
	assertNextToken(t, l, String, "\"name\"", 1, 15, 15)
	assertNextToken(t, l, ',', ",", 1, 21, 21)
	assertNextToken(t, l, Ident, "timeout", 1, 23, 23)
	assertNextToken(t, l, '=', "=", 1, 30, 30)
	assertNextToken(t, l, Int, "10", 1, 31, 31)
	assertNextToken(t, l, ',', ",", 1, 33, 33)
	assertNextToken(t, l, Ident, "args", 1, 35, 35)
	assertNextToken(t, l, '=', "=", 1, 39, 39)
	assertNextToken(t, l, Ident, "CONFIG", 1, 40, 40)
	assertNextToken(t, l, '.', ".", 1, 46, 46)
	assertNextToken(t, l, Ident, "ARGS", 1, 47, 47)
	assertNextToken(t, l, ')', ")", 1, 51, 51)
	assertNextToken(t, l, ':', ":", 1, 52, 52)
}

const inputFunction = `
python_library(
    name = 'lib',
    srcs = [
        'lib1.py',
        'lib2.py',
    ],
)
`

func TestMoreComplexFunction(t *testing.T) {
	l := newLexer(strings.NewReader(inputFunction))
	assertNextToken(t, l, Ident, "python_library", 2, 1, 2)
	assertNextToken(t, l, '(', "(", 2, 15, 16)
	assertNextToken(t, l, Ident, "name", 3, 5, 22)
	assertNextToken(t, l, '=', "=", 3, 10, 27)
	assertNextToken(t, l, String, "\"lib\"", 3, 12, 29)
	assertNextToken(t, l, ',', ",", 3, 17, 34)
	assertNextToken(t, l, Ident, "srcs", 4, 5, 40)
	assertNextToken(t, l, '=', "=", 4, 10, 45)
	assertNextToken(t, l, '[', "[", 4, 12, 47)
	assertNextToken(t, l, String, "\"lib1.py\"", 5, 9, 57)
	assertNextToken(t, l, ',', ",", 5, 18, 66)
	assertNextToken(t, l, String, "\"lib2.py\"", 6, 9, 76)
	assertNextToken(t, l, ',', ",", 6, 18, 85)
	assertNextToken(t, l, ']', "]", 7, 5, 91)
	assertNextToken(t, l, ',', ",", 7, 6, 92)
	assertNextToken(t, l, ')', ")", 8, 1, 94)
}

const multiUnindent = `
for y in x:
    for z in y:
        for a in z:
            pass
`

func TestMultiUnindent(t *testing.T) {
	l := newLexer(strings.NewReader(multiUnindent))
	assertNextToken(t, l, Ident, "for", 2, 1, 2)
	assertNextToken(t, l, Ident, "y", 2, 5, 6)
	assertNextToken(t, l, Ident, "in", 2, 7, 8)
	assertNextToken(t, l, Ident, "x", 2, 10, 11)
	assertNextToken(t, l, ':', ":", 2, 11, 12)
	assertNextToken(t, l, EOL, "", 2, 12, 13)
	assertNextToken(t, l, Ident, "for", 3, 5, 18)
	assertNextToken(t, l, Ident, "z", 3, 9, 22)
	assertNextToken(t, l, Ident, "in", 3, 11, 24)
	assertNextToken(t, l, Ident, "y", 3, 14, 27)
	assertNextToken(t, l, ':', ":", 3, 15, 28)
	assertNextToken(t, l, EOL, "", 3, 16, 29)
	assertNextToken(t, l, Ident, "for", 4, 9, 38)
	assertNextToken(t, l, Ident, "a", 4, 13, 42)
	assertNextToken(t, l, Ident, "in", 4, 15, 44)
	assertNextToken(t, l, Ident, "z", 4, 18, 47)
	assertNextToken(t, l, ':', ":", 4, 19, 48)
	assertNextToken(t, l, EOL, "", 4, 20, 49)
	assertNextToken(t, l, Ident, "pass", 5, 13, 62)
	assertNextToken(t, l, EOL, "", 6, 1, 67)
	assertNextToken(t, l, Unindent, "", 6, 1, 67)
	assertNextToken(t, l, Unindent, "", 6, 1, 67)
	assertNextToken(t, l, Unindent, "", 6, 1, 67)
}

const multiLineFunctionArgs = `
def test(name='name', timeout=10,
         args=CONFIG.ARGS):
    pass
`

func TestMultiLineFunctionArgs(t *testing.T) {
	l := newLexer(strings.NewReader(multiLineFunctionArgs))
	assertNextToken(t, l, Ident, "def", 2, 1, 2)
	assertNextToken(t, l, Ident, "test", 2, 5, 6)
	assertNextToken(t, l, '(', "(", 2, 9, 10)
	assertNextToken(t, l, Ident, "name", 2, 10, 11)
	assertNextToken(t, l, '=', "=", 2, 14, 15)
	assertNextToken(t, l, String, "\"name\"", 2, 15, 16)
	assertNextToken(t, l, ',', ",", 2, 21, 22)
	assertNextToken(t, l, Ident, "timeout", 2, 23, 24)
	assertNextToken(t, l, '=', "=", 2, 30, 31)
	assertNextToken(t, l, Int, "10", 2, 31, 32)
	assertNextToken(t, l, ',', ",", 2, 33, 34)
	assertNextToken(t, l, Ident, "args", 3, 10, 45)
	assertNextToken(t, l, '=', "=", 3, 14, 49)
	assertNextToken(t, l, Ident, "CONFIG", 3, 15, 50)
	assertNextToken(t, l, '.', ".", 3, 21, 56)
	assertNextToken(t, l, Ident, "ARGS", 3, 22, 57)
	assertNextToken(t, l, ')', ")", 3, 26, 61)
	assertNextToken(t, l, ':', ":", 3, 27, 62)
	assertNextToken(t, l, EOL, "", 3, 28, 63)
	assertNextToken(t, l, Ident, "pass", 4, 5, 68)
	assertNextToken(t, l, EOL, "", 5, 1, 73)
	assertNextToken(t, l, Unindent, "", 5, 1, 73)
}

func TestComparisonOperator(t *testing.T) {
	l := newLexer(strings.NewReader("x = y == z"))
	assertNextToken(t, l, Ident, "x", 1, 1, 1)
	assertNextToken(t, l, '=', "=", 1, 3, 3)
	assertNextToken(t, l, Ident, "y", 1, 5, 5)
	assertNextToken(t, l, LexOperator, "==", 1, 7, 7)
}

const blankLinesInFunction = `
def x():
    """test"""

    return 42
`

func TestBlankLinesInFunction(t *testing.T) {
	l := newLexer(strings.NewReader(blankLinesInFunction))
	assertNextToken(t, l, Ident, "def", 2, 1, 2)
	assertNextToken(t, l, Ident, "x", 2, 5, 6)
	assertNextToken(t, l, '(', "(", 2, 6, 7)
	assertNextToken(t, l, ')', ")", 2, 7, 8)
	assertNextToken(t, l, ':', ":", 2, 8, 9)
	assertNextToken(t, l, EOL, "", 2, 9, 10)
	assertNextToken(t, l, String, "\"test\"", 3, 5, 15)
	assertNextToken(t, l, EOL, "", 4, 1, 26)
	assertNextToken(t, l, Ident, "return", 5, 5, 31)
	assertNextToken(t, l, Int, "42", 5, 12, 38)
	assertNextToken(t, l, EOL, "", 6, 1, 41)
	assertNextToken(t, l, Unindent, "", 6, 1, 41)
}

const commentsAndEOLs = `
pass

# something

`

func TestCommentsAndEOLs(t *testing.T) {
	l := newLexer(strings.NewReader(commentsAndEOLs))
	assertNextToken(t, l, Ident, "pass", 2, 1, 2)
	assertNextToken(t, l, EOL, "", 3, 1, 7)
	assertNextToken(t, l, EOF, "", 6, 1, 21)
}

// This is a much-simplified version of the true motivating case.
const unevenIndent = `
def x():
    if True:
            pass
    return
`

func TestUnevenUnindent(t *testing.T) {
	l := newLexer(strings.NewReader(unevenIndent))
	assertNextToken(t, l, Ident, "def", 2, 1, 2)
	assertNextToken(t, l, Ident, "x", 2, 5, 6)
	assertNextToken(t, l, '(', "(", 2, 6, 7)
	assertNextToken(t, l, ')', ")", 2, 7, 8)
	assertNextToken(t, l, ':', ":", 2, 8, 9)
	assertNextToken(t, l, EOL, "", 2, 9, 10)
	assertNextToken(t, l, Ident, "if", 3, 5, 15)
	assertNextToken(t, l, Ident, "True", 3, 8, 18)
	assertNextToken(t, l, ':', ":", 3, 12, 22)
	assertNextToken(t, l, EOL, "", 3, 13, 23)
	assertNextToken(t, l, Ident, "pass", 4, 13, 36)
	assertNextToken(t, l, EOL, "", 5, 1, 41)
	assertNextToken(t, l, Unindent, "", 5, 5, 45)
	assertNextToken(t, l, Ident, "return", 5, 5, 45)
	assertNextToken(t, l, EOL, "", 6, 1, 52)
	assertNextToken(t, l, Unindent, "", 6, 1, 52)
	assertNextToken(t, l, EOF, "", 6, 1, 52)
}

func TestCRLF(t *testing.T) {
	l := newLexer(strings.NewReader("package()\r\nsubinclude()\r\n"))
	assertNextToken(t, l, Ident, "package", 1, 1, 1)
	assertNextToken(t, l, '(', "(", 1, 8, 8)
	assertNextToken(t, l, ')', ")", 1, 9, 9)
	assertNextToken(t, l, EOL, "", 1, 11, 11)
	assertNextToken(t, l, Ident, "subinclude", 2, 1, 12)
	assertNextToken(t, l, '(', "(", 2, 11, 22)
	assertNextToken(t, l, ')', ")", 2, 12, 23)
	assertNextToken(t, l, EOL, "", 2, 14, 25)
}

func TestOctal(t *testing.T) {
	l := newLexer(strings.NewReader("0o604 0604"))
	assertNextToken(t, l, Int, "0604", 1, 1, 1)
	assertNextToken(t, l, Int, "0604", 1, 7, 7)
}

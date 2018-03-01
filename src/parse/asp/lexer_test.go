package asp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertToken(t *testing.T, tok Token, tokenType rune, value string, line, column, offset int) {
	assert.EqualValues(t, tokenType, tok.Type, "incorrect type")
	assert.Equal(t, value, tok.Value, "incorrect value")
	assert.Equal(t, line, tok.Pos.Line, "incorrect line")
	assert.Equal(t, column, tok.Pos.Column, "incorrect column")
	assert.Equal(t, offset, tok.Pos.Offset, "incorrect offset")
}

func TestLexBasic(t *testing.T) {
	l := newLexer(strings.NewReader("hello world"))
	assertToken(t, l.Next(), Ident, "hello", 1, 1, 1)
	assertToken(t, l.Peek(), Ident, "world", 1, 7, 7)
	assertToken(t, l.Next(), Ident, "world", 1, 7, 7)
	assertToken(t, l.Next(), EOL, "", 1, 12, 12)
	assertToken(t, l.Next(), EOF, "", 2, 1, 13)
}

func TestLexMultiline(t *testing.T) {
	l := newLexer(strings.NewReader("hello\nworld\n"))
	assertToken(t, l.Next(), Ident, "hello", 1, 1, 1)
	assertToken(t, l.Next(), EOL, "", 1, 6, 6)
	assertToken(t, l.Next(), Ident, "world", 2, 1, 7)
	assertToken(t, l.Next(), EOL, "", 2, 6, 12)
	assertToken(t, l.Next(), EOF, "", 3, 1, 13)
}

const testFunction = `
def func(x):
    pass
`

func TestLexFunction(t *testing.T) {
	l := newLexer(strings.NewReader(testFunction))
	assertToken(t, l.Next(), Ident, "def", 2, 1, 2)
	assertToken(t, l.Next(), Ident, "func", 2, 5, 6)
	assertToken(t, l.Next(), '(', "(", 2, 9, 10)
	assertToken(t, l.Next(), Ident, "x", 2, 10, 11)
	assertToken(t, l.Next(), ')', ")", 2, 11, 12)
	assertToken(t, l.Next(), ':', ":", 2, 12, 13)
	assertToken(t, l.Next(), EOL, "", 2, 13, 14)
	assertToken(t, l.Next(), Ident, "pass", 3, 5, 19)
	assertToken(t, l.Next(), EOL, "", 4, 1, 23)
	assertToken(t, l.Next(), Unindent, "", 4, 1, 24)
	assertToken(t, l.Next(), EOF, "", 4, 1, 24)
}

func TestLexUnicode(t *testing.T) {
	l := newLexer(strings.NewReader("懂了吗 你愁脸 有没有"))
	assertToken(t, l.Next(), Ident, "懂了吗", 1, 1, 1)
	assertToken(t, l.Next(), Ident, "你愁脸", 1, 11, 11)
	assertToken(t, l.Next(), Ident, "有没有", 1, 21, 21)
	assertToken(t, l.Next(), EOL, "", 1, 30, 30)
	assertToken(t, l.Next(), EOF, "", 2, 1, 31)
}

func TestLexString(t *testing.T) {
	l := newLexer(strings.NewReader(`x = "hello world"`))
	assertToken(t, l.Next(), Ident, "x", 1, 1, 1)
	assertToken(t, l.Next(), '=', "=", 1, 3, 3)
	assertToken(t, l.Next(), String, "\"hello world\"", 1, 5, 5)
	assertToken(t, l.Next(), EOL, "", 1, 18, 18)
	assertToken(t, l.Next(), EOF, "", 2, 1, 19)
}

func TestLexStringEscape(t *testing.T) {
	l := newLexer(strings.NewReader(`x = '\n\\'`))
	assertToken(t, l.Next(), Ident, "x", 1, 1, 1)
	assertToken(t, l.Next(), '=', "=", 1, 3, 3)
	assertToken(t, l.Next(), String, "\"\n\\\"", 1, 5, 5)
	assertToken(t, l.Next(), EOL, "", 1, 11, 11)
	assertToken(t, l.Next(), EOF, "", 2, 1, 12)
}

func TestLexStringEscape2(t *testing.T) {
	l := newLexer(strings.NewReader(`'echo -n "import \( \";'`))
	assertToken(t, l.Next(), String, `"echo -n "import \( ";"`, 1, 1, 1)
	assertToken(t, l.Next(), EOL, "", 1, 25, 25)
	assertToken(t, l.Next(), EOF, "", 2, 1, 26)
}

func TestLexRawString(t *testing.T) {
	l := newLexer(strings.NewReader(`x = r'\n\\'`))
	assertToken(t, l.Next(), Ident, "x", 1, 1, 1)
	assertToken(t, l.Next(), '=', "=", 1, 3, 3)
	assertToken(t, l.Next(), String, `"\n\\"`, 1, 5, 5)
	assertToken(t, l.Next(), EOL, "", 1, 12, 12)
	assertToken(t, l.Next(), EOF, "", 2, 1, 13)
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
	assertToken(t, l.Next(), Ident, "x", 1, 1, 1)
	assertToken(t, l.Next(), '=', "=", 1, 3, 3)
	assertToken(t, l.Next(), String, expectedMultilineString, 1, 5, 5)
	assertToken(t, l.Next(), EOL, "", 4, 4, 26)
	assertToken(t, l.Next(), EOF, "", 5, 1, 27)
}

func TestLexAttributeAccess(t *testing.T) {
	l := newLexer(strings.NewReader(`x.call(y)`))
	assertToken(t, l.Next(), Ident, "x", 1, 1, 1)
	assertToken(t, l.Next(), '.', ".", 1, 2, 2)
	assertToken(t, l.Next(), Ident, "call", 1, 3, 3)
	assertToken(t, l.Next(), '(', "(", 1, 7, 7)
	assertToken(t, l.Next(), Ident, "y", 1, 8, 8)
	assertToken(t, l.Next(), ')', ")", 1, 9, 9)
	assertToken(t, l.Next(), EOL, "", 1, 10, 10)
	assertToken(t, l.Next(), EOF, "", 2, 1, 11)
}

func TestLexFunctionArgs(t *testing.T) {
	l := newLexer(strings.NewReader(`def test(name='name', timeout=10, args=CONFIG.ARGS):`))
	assertToken(t, l.Next(), Ident, "def", 1, 1, 1)
	assertToken(t, l.Next(), Ident, "test", 1, 5, 5)
	assertToken(t, l.Next(), '(', "(", 1, 9, 9)
	assertToken(t, l.Next(), Ident, "name", 1, 10, 10)
	assertToken(t, l.Next(), '=', "=", 1, 14, 14)
	assertToken(t, l.Next(), String, "\"name\"", 1, 15, 15)
	assertToken(t, l.Next(), ',', ",", 1, 21, 21)
	assertToken(t, l.Next(), Ident, "timeout", 1, 23, 23)
	assertToken(t, l.Next(), '=', "=", 1, 30, 30)
	assertToken(t, l.Next(), Int, "10", 1, 31, 31)
	assertToken(t, l.Next(), ',', ",", 1, 33, 33)
	assertToken(t, l.Next(), Ident, "args", 1, 35, 35)
	assertToken(t, l.Next(), '=', "=", 1, 39, 39)
	assertToken(t, l.Next(), Ident, "CONFIG", 1, 40, 40)
	assertToken(t, l.Next(), '.', ".", 1, 46, 46)
	assertToken(t, l.Next(), Ident, "ARGS", 1, 47, 47)
	assertToken(t, l.Next(), ')', ")", 1, 51, 51)
	assertToken(t, l.Next(), ':', ":", 1, 52, 52)
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
	assertToken(t, l.Next(), Ident, "python_library", 2, 1, 2)
	assertToken(t, l.Next(), '(', "(", 2, 15, 16)
	assertToken(t, l.Next(), Ident, "name", 3, 5, 22)
	assertToken(t, l.Next(), '=', "=", 3, 10, 27)
	assertToken(t, l.Next(), String, "\"lib\"", 3, 12, 29)
	assertToken(t, l.Next(), ',', ",", 3, 17, 34)
	assertToken(t, l.Next(), Ident, "srcs", 4, 5, 40)
	assertToken(t, l.Next(), '=', "=", 4, 10, 45)
	assertToken(t, l.Next(), '[', "[", 4, 12, 47)
	assertToken(t, l.Next(), String, "\"lib1.py\"", 5, 9, 57)
	assertToken(t, l.Next(), ',', ",", 5, 18, 66)
	assertToken(t, l.Next(), String, "\"lib2.py\"", 6, 9, 76)
	assertToken(t, l.Next(), ',', ",", 6, 18, 85)
	assertToken(t, l.Next(), ']', "]", 7, 5, 91)
	assertToken(t, l.Next(), ',', ",", 7, 6, 92)
	assertToken(t, l.Next(), ')', ")", 8, 1, 94)
}

const multiUnindent = `
for y in x:
    for z in y:
        for a in z:
            pass
`

func TestMultiUnindent(t *testing.T) {
	l := newLexer(strings.NewReader(multiUnindent))
	assertToken(t, l.Next(), Ident, "for", 2, 1, 2)
	assertToken(t, l.Next(), Ident, "y", 2, 5, 6)
	assertToken(t, l.Next(), Ident, "in", 2, 7, 8)
	assertToken(t, l.Next(), Ident, "x", 2, 10, 11)
	assertToken(t, l.Next(), ':', ":", 2, 11, 12)
	assertToken(t, l.Next(), EOL, "", 2, 12, 13)
	assertToken(t, l.Next(), Ident, "for", 3, 5, 18)
	assertToken(t, l.Next(), Ident, "z", 3, 9, 22)
	assertToken(t, l.Next(), Ident, "in", 3, 11, 24)
	assertToken(t, l.Next(), Ident, "y", 3, 14, 27)
	assertToken(t, l.Next(), ':', ":", 3, 15, 28)
	assertToken(t, l.Next(), EOL, "", 3, 16, 29)
	assertToken(t, l.Next(), Ident, "for", 4, 9, 38)
	assertToken(t, l.Next(), Ident, "a", 4, 13, 42)
	assertToken(t, l.Next(), Ident, "in", 4, 15, 44)
	assertToken(t, l.Next(), Ident, "z", 4, 18, 47)
	assertToken(t, l.Next(), ':', ":", 4, 19, 48)
	assertToken(t, l.Next(), EOL, "", 4, 20, 49)
	assertToken(t, l.Next(), Ident, "pass", 5, 13, 62)
	assertToken(t, l.Next(), EOL, "", 6, 1, 66)
	assertToken(t, l.Next(), Unindent, "", 6, 1, 67)
	assertToken(t, l.Next(), Unindent, "", 6, 1, 67)
	assertToken(t, l.Next(), Unindent, "", 6, 1, 67)
}

const multiLineFunctionArgs = `
def test(name='name', timeout=10,
         args=CONFIG.ARGS):
    pass
`

func TestMultiLineFunctionArgs(t *testing.T) {
	l := newLexer(strings.NewReader(multiLineFunctionArgs))
	assertToken(t, l.Next(), Ident, "def", 2, 1, 2)
	assertToken(t, l.Next(), Ident, "test", 2, 5, 6)
	assertToken(t, l.Next(), '(', "(", 2, 9, 10)
	assertToken(t, l.Next(), Ident, "name", 2, 10, 11)
	assertToken(t, l.Next(), '=', "=", 2, 14, 15)
	assertToken(t, l.Next(), String, "\"name\"", 2, 15, 16)
	assertToken(t, l.Next(), ',', ",", 2, 21, 22)
	assertToken(t, l.Next(), Ident, "timeout", 2, 23, 24)
	assertToken(t, l.Next(), '=', "=", 2, 30, 31)
	assertToken(t, l.Next(), Int, "10", 2, 31, 32)
	assertToken(t, l.Next(), ',', ",", 2, 33, 34)
	assertToken(t, l.Next(), Ident, "args", 3, 10, 45)
	assertToken(t, l.Next(), '=', "=", 3, 14, 49)
	assertToken(t, l.Next(), Ident, "CONFIG", 3, 15, 50)
	assertToken(t, l.Next(), '.', ".", 3, 21, 56)
	assertToken(t, l.Next(), Ident, "ARGS", 3, 22, 57)
	assertToken(t, l.Next(), ')', ")", 3, 26, 61)
	assertToken(t, l.Next(), ':', ":", 3, 27, 62)
	assertToken(t, l.Next(), EOL, "", 3, 28, 63)
	assertToken(t, l.Next(), Ident, "pass", 4, 5, 68)
	assertToken(t, l.Next(), EOL, "", 5, 1, 72)
	assertToken(t, l.Next(), Unindent, "", 5, 1, 73)
}

func TestComparisonOperator(t *testing.T) {
	l := newLexer(strings.NewReader("x = y == z"))
	assertToken(t, l.Next(), Ident, "x", 1, 1, 1)
	assertToken(t, l.Next(), '=', "=", 1, 3, 3)
	assertToken(t, l.Next(), Ident, "y", 1, 5, 5)
	assertToken(t, l.Next(), LexOperator, "==", 1, 7, 7)
}

const blankLinesInFunction = `
def x():
    """test"""

    return 42
`

func TestBlankLinesInFunction(t *testing.T) {
	l := newLexer(strings.NewReader(blankLinesInFunction))
	assertToken(t, l.Next(), Ident, "def", 2, 1, 2)
	assertToken(t, l.Next(), Ident, "x", 2, 5, 6)
	assertToken(t, l.Next(), '(', "(", 2, 6, 7)
	assertToken(t, l.Next(), ')', ")", 2, 7, 8)
	assertToken(t, l.Next(), ':', ":", 2, 8, 9)
	assertToken(t, l.Next(), EOL, "", 2, 9, 10)
	assertToken(t, l.Next(), String, "\"test\"", 3, 5, 15)
	assertToken(t, l.Next(), EOL, "", 4, 1, 26)
	assertToken(t, l.Next(), Ident, "return", 5, 5, 31)
	assertToken(t, l.Next(), Int, "42", 5, 12, 38)
	assertToken(t, l.Next(), EOL, "", 6, 1, 40)
	assertToken(t, l.Next(), Unindent, "", 6, 1, 41)
}

const commentsAndEOLs = `
pass

# something

`

func TestCommentsAndEOLs(t *testing.T) {
	l := newLexer(strings.NewReader(commentsAndEOLs))
	assertToken(t, l.Next(), Ident, "pass", 2, 1, 2)
	assertToken(t, l.Next(), EOL, "", 3, 1, 7)
	assertToken(t, l.Next(), EOF, "", 6, 1, 21)
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
	assertToken(t, l.Next(), Ident, "def", 2, 1, 2)
	assertToken(t, l.Next(), Ident, "x", 2, 5, 6)
	assertToken(t, l.Next(), '(', "(", 2, 6, 7)
	assertToken(t, l.Next(), ')', ")", 2, 7, 8)
	assertToken(t, l.Next(), ':', ":", 2, 8, 9)
	assertToken(t, l.Next(), EOL, "", 2, 9, 10)
	assertToken(t, l.Next(), Ident, "if", 3, 5, 15)
	assertToken(t, l.Next(), Ident, "True", 3, 8, 18)
	assertToken(t, l.Next(), ':', ":", 3, 12, 22)
	assertToken(t, l.Next(), EOL, "", 3, 13, 23)
	assertToken(t, l.Next(), Ident, "pass", 4, 13, 36)
	assertToken(t, l.Next(), EOL, "", 5, 5, 40)
	assertToken(t, l.Next(), Unindent, "", 5, 5, 45)
	assertToken(t, l.Next(), Ident, "return", 5, 5, 45)
	assertToken(t, l.Next(), EOL, "", 6, 1, 51)
	assertToken(t, l.Next(), Unindent, "", 6, 1, 52)
	assertToken(t, l.Next(), EOF, "", 6, 1, 52)
}

const implicitStringConcatenation = `
str('testing that we can carry these '
    'over multiple lines')
`

func TestImplicitStringConcatenation(t *testing.T) {
	l := newLexer(strings.NewReader(implicitStringConcatenation))
	assertToken(t, l.Next(), Ident, "str", 2, 1, 2)
	assertToken(t, l.Next(), '(', "(", 2, 4, 5)
	assertToken(t, l.Next(), String, `"testing that we can carry these over multiple lines"`, 2, 5, 6)
}

func TestImplicitStringConcatenationOnlyHappensInsideBraces(t *testing.T) {
	l := newLexer(strings.NewReader("'hello' 'world'"))
	assertToken(t, l.Next(), String, `"hello"`, 1, 1, 1)
	assertToken(t, l.Next(), String, `"world"`, 1, 9, 9)
}

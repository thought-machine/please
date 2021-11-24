package asp

import (
	"bufio"
	"io"
	"unicode"
	"unicode/utf8"
)

// Token types.
const (
	EOF = -(iota + 1)
	Ident
	Int
	String
	LexOperator
	EOL
	Unindent
)

// A Token describes each individual lexical element emitted by the lexer.
type Token struct {
	// Type of token. If > 0 this is the literal character value; if < 0 it is one of the types above.
	Type rune
	// The literal text of the token. Strings are lightly normalised to always be surrounded by quotes (but only one).
	Value string
	// The position in the input that the token occurred at.
	Pos Position
}

// String implements the fmt.Stringer interface
func (tok Token) String() string {
	if tok.Value != "" {
		return tok.Value
	}
	return reverseSymbol(tok.Type)
}

// EndPos returns the end position of a token
func (tok Token) EndPos() Position {
	end := tok.Pos
	end.Offset += len(tok.Value)
	end.Column += len(tok.Value)

	return end
}

type namer interface {
	Name() string
}

// NameOfReader returns a name for the given reader, if one can be determined.
func NameOfReader(r io.Reader) string {
	if n, ok := r.(namer); ok {
		return n.Name()
	}
	return ""
}

// newLexer creates a new lex instance.
func newLexer(r io.Reader) *lex {
	l := &lex{
		reader:   bufio.NewReader(r),
		filename: NameOfReader(r),
		indents:  []int{0},
	}
	l.nextRune, _, _ = l.reader.ReadRune()
	l.readNextByte()
	l.Next() // Initial value is zero, this forces it to populate itself.
	// Discard any leading newlines, they are just an annoyance.
	for l.Peek().Type == EOL {
		l.Next()
	}
	return l
}

// A lex is a lexer for a single BUILD file.
type lex struct {
	// The reader we're lexing
	reader *bufio.Reader
	// the current and next bytes from pos
	currentRune, nextRune rune
	// Positional information about where we are in the file
	pos, line, col int
	// The current level of indentation we're on in the file
	indent int
	// The next token. We always look one token ahead in order to facilitate both Peek() and Next().
	next Token
	// The name of the file we're parsing. Can be unset if we're parsing a non-file reader.
	filename string
	// Used to track how many braces we're within.
	braces int
	// Pending unindent tokens. This is a bit yuck but means the parser doesn't need to
	// concern itself about indentation.
	unindents int
	// Current levels of indentation
	indents []int
	// Remember whether the last token we output was an end-of-line so we don't emit multiple in sequence.
	lastEOL bool
}

// readNextByte updates the current and next byte fields by consuming from the underlying reader
func (l *lex) readNextByte() {
	var err error
	l.currentRune = l.nextRune
	l.nextRune, _, err = l.reader.ReadRune()

	if err != nil {
		if err != io.EOF {
			fail(Position{Filename: l.filename, Offset: l.pos, Line: l.line, Column: l.col}, err.Error())
		}

		if l.currentRune != '\n' {
			// If the file doesn't end in a newline, we will reject it with an "unexpected end of file"
			// error. That's a bit crap so quietly fix it up here.
			l.nextRune = '\n'
		} else {
			l.nextRune = 0 // This essentially null terminates the file which is useful later on
		}
	}
}

// advance reads moves to the next position in the reader, updating nextRune and currentRune
func (l *lex) advance() {
	l.col++
	l.pos++
	l.readNextByte()
}

// reverseSymbol looks up a symbol's name from the lexer.
func reverseSymbol(sym rune) string {
	switch sym {
	case EOF:
		return "end of file"
	case Ident:
		return "identifier"
	case Int:
		return "integer"
	case String:
		return "string"
	case LexOperator:
		return "operator"
	case EOL:
		return "end of line"
	case Unindent:
		return "unindent"
	}
	return string(sym) // literal character
}

// reverseSymbols looks up a series of symbol's names from the lexer.
func reverseSymbols(syms []rune) []string {
	ret := make([]string, len(syms))
	for i, sym := range syms {
		ret[i] = reverseSymbol(sym)
	}
	return ret
}

// Peek at the next token
func (l *lex) Peek() Token {
	return l.next
}

// Next consumes and returns the next token.
func (l *lex) Next() Token {
	ret := l.next
	l.next = l.nextToken()
	l.lastEOL = l.next.Type == EOL || l.next.Type == Unindent
	return ret
}

// AssignFollows is a hack to do extra lookahead which makes it easier to parse
// named call arguments. It returns true if the token after next is an assign operator.
func (l *lex) AssignFollows() bool {
	l.stripSpaces()
	return l.currentRune == '=' && l.nextRune != '='
}

func (l *lex) stripSpaces() {
	for l.currentRune == ' ' {
		l.advance()
	}
}

// nextToken consumes and returns the next token.
func (l *lex) nextToken() Token {
	l.stripSpaces()
	pos := Position{
		Filename: l.filename,
		// These are all 1-indexed for niceness.
		Offset: l.pos + 1,
		Line:   l.line + 1,
		Column: l.col + 1,
	}
	if l.unindents > 0 {
		l.unindents--
		return Token{Type: Unindent, Pos: pos}
	}
	next := l.currentRune
	rawString := next == 'r' && (l.nextRune == '"' || l.nextRune == '\'')
	fString := next == 'f' && (l.nextRune == '"' || l.nextRune == '\'')
	if rawString || fString {
		l.advance()
		next = l.currentRune
	} else if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') || next == '_' || next >= utf8.RuneSelf {
		return l.consumeIdent(pos)
	}
	l.advance()
	switch next {
	case 0:
		// End of file (we null terminate it above so this is easy to spot)
		return Token{Type: EOF, Pos: pos}
	case '\r':
		return l.nextToken()
	case '\n':
		// End of line, read indent to next non-space character
		lastIndent := l.indent
		l.line++
		l.col = 0
		indent := 0
		for l.currentRune == ' ' {
			l.advance()
			indent++
		}
		if l.currentRune == '\n' {
			return l.nextToken()
		}
		if l.braces == 0 {
			l.indent = indent
		}
		if lastIndent > l.indent && l.braces == 0 {
			pos.Line++ // Works better if it's at the new position
			pos.Column = l.col + 1
			for l.indents[len(l.indents)-1] > l.indent {
				l.unindents++
				l.indents = l.indents[:len(l.indents)-1]
			}
			if l.indent != l.indents[len(l.indents)-1] {
				fail(pos, "Unexpected indent")
			}
		} else if lastIndent != l.indent {
			l.indents = append(l.indents, l.indent)
		}
		if l.braces == 0 && !l.lastEOL {
			return Token{Type: EOL, Pos: pos}
		}
		return l.nextToken()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return l.consumeInteger(next, pos)
	case '"', '\'':
		// String literal, consume to end.
		return l.consumePossiblyTripleQuotedString(next, pos, rawString, fString)
	case '(', '[', '{':
		l.braces++
		return Token{Type: next, Value: string(next), Pos: pos}
	case ')', ']', '}':
		if l.braces > 0 { // Don't let it go negative, it fouls things up
			l.braces--
		}
		return Token{Type: next, Value: string(next), Pos: pos}
	case '=', '!', '+', '<', '>':
		// Look ahead one byte to see if this is an augmented assignment or comparison.
		if l.currentRune == '=' {
			l.advance()
			return Token{Type: LexOperator, Value: string([]rune{next, '='}), Pos: pos}
		}
		fallthrough
	case ',', '.', '%', '*', '|', '&', ':', '/':
		return Token{Type: next, Value: string(next), Pos: pos}
	case '#':
		// Comment character, consume to end of line.
		for l.currentRune != '\n' && l.currentRune != 0 {
			l.advance()
		}
		return l.nextToken() // Comments aren't tokens themselves.
	case '-':
		// We lex unary - with the integer if possible.
		if l.currentRune >= '0' && l.currentRune <= '9' {
			return l.consumeInteger(next, pos)
		}
		return Token{Type: next, Value: string(next), Pos: pos}
	case '\t':
		fail(pos, "Tabs are not permitted in BUILD files, use space-based indentation instead")
	default:
		fail(pos, "Unknown symbol %c", next)
	}
	panic("unreachable")
}

// consumeInteger consumes all characters until the end of an integer literal is reached.
func (l *lex) consumeInteger(initial rune, pos Position) Token {
	value := make([]byte, 1, 10)

	value[0] = byte(initial)
	for next := l.currentRune; next >= '0' && next <= '9'; next = l.currentRune {
		l.advance()
		value = append(value, byte(next))
	}
	return Token{Type: Int, Value: string(value), Pos: pos}
}

// consumePossiblyTripleQuotedString consumes all characters until the end of a string token.
func (l *lex) consumePossiblyTripleQuotedString(quote rune, pos Position, raw, fString bool) Token {
	if l.currentRune == quote && l.nextRune == quote {
		l.advance() // Jump over initial quote
		l.advance()
		return l.consumeString(quote, pos, true, raw, fString)
	}
	return l.consumeString(quote, pos, false, raw, fString)
}

// consumeString consumes all characters until the end of a string literal is reached.
func (l *lex) consumeString(quote rune, pos Position, multiline, raw, fString bool) Token {
	value := make([]rune, 1, 100) // 100 chars is typically enough for a single string literal.
	value[0] = '"'
	escaped := false
	for {
		next := l.currentRune
		l.advance()
		if escaped {
			if next == 'n' {
				value = append(value, '\n')
			} else if next == '\n' && multiline {
				l.line++
				l.col = 0
			} else if next == '\\' || next == '\'' || next == '"' {
				value = append(value, next)
			} else {
				value = append(value, '\\', next)
			}
			escaped = false
			continue
		}
		switch next {
		case quote:
			if !multiline || (l.currentRune == quote && l.nextRune == quote) {
				value = append(value, '"')
				if multiline {
					l.advance()
					l.advance()
				}
				token := Token{Type: String, Value: string(value), Pos: pos}
				if fString {
					token.Value = "f" + token.Value
				}
				return token
			}
			value = append(value, next)
		case '\n':
			if multiline {
				l.line++
				l.col = 0
				value = append(value, next)
				continue
			}
			fallthrough
		case 0:
			fail(pos, "Unterminated string literal")
		case '\\':
			if !raw {
				escaped = true
				continue
			}
			fallthrough
		default:
			value = append(value, next)
		}
	}
}

// consumeIdent consumes all characters of an identifier.
func (l *lex) consumeIdent(pos Position) Token {
	s := make([]rune, 0, 100)
	for {
		c := l.currentRune

		if c == ' ' {
			l.advance()
			// End of identifier, but no unconsuming needed.
			return Token{Type: Ident, Value: string(s), Pos: pos}
		} else if c == '_' || unicode.IsLetter(c) || unicode.IsDigit(c) {
			l.advance()
			s = append(s, c)
		} else {
			// End of identifier.
			return Token{Type: Ident, Value: string(s), Pos: pos}
		}
	}
}

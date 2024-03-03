package asp

import (
	"io"
	"runtime/debug"
	"strconv"
	"strings"
)

// keywords are the list of reserved keywords in the language. They can't be assigned to.
// Not all of these have meaning in the build language (and many never will), but they are
// reserved in Python and for practical reasons it is useful to remain a subset of Python.
var keywords = map[string]struct{}{
	"False":    {},
	"None":     {},
	"True":     {},
	"and":      {},
	"as":       {},
	"assert":   {},
	"break":    {},
	"class":    {},
	"continue": {},
	"def":      {},
	"del":      {},
	"elif":     {},
	"else":     {},
	"except":   {},
	"finally":  {},
	"for":      {},
	"from":     {},
	"global":   {},
	"if":       {},
	"import":   {},
	"in":       {},
	"is":       {},
	"lambda":   {},
	"nonlocal": {},
	"not":      {},
	"or":       {},
	"pass":     {},
	"raise":    {},
	"return":   {},
	"try":      {},
	"while":    {},
	"with":     {},
	"yield":    {},
}

type parser struct {
	l      *lex
	endPos Position
}

// parseFileInput is the only external entry point to this class, it parses a file into a FileInput structure.
func parseFileInput(r io.Reader) (input *FileInput, err error) {
	input = &FileInput{}
	// The rest of the parser functions signal unhappiness by panicking, we
	// recover any such failures here and convert to an error.
	defer func() {
		if r := recover(); r != nil {
			log.Debugf("error parsing build file: %v \n%v", r, string(debug.Stack()))
			err = r.(error)
		}
	}()

	p := &parser{l: newLexer(r)}
	for tok := p.l.Peek(); tok.Type != EOF; tok = p.l.Peek() {
		input.Statements = append(input.Statements, p.parseStatement())
	}
	return input, nil
}

func (p *parser) assert(condition bool, pos Token, message string, args ...interface{}) {
	if !condition {
		p.fail(pos, message, args...)
	}
}

func (p *parser) assertTokenType(tok Token, expectedType rune) {
	if tok.Type != expectedType {
		p.fail(tok, "unexpected token %s, expected %s", tok, reverseSymbol(expectedType))
	}
}

func (p *parser) next(expectedType rune) Token {
	tok := p.l.Next()
	p.assertTokenType(tok, expectedType)
	return tok
}

func (p *parser) nextv(expectedValue string) Token {
	tok := p.l.Next()
	if tok.Value != expectedValue {
		p.fail(tok, "unexpected token %s, expected %s", tok, expectedValue)
	}
	return tok
}

func (p *parser) optional(option rune) bool {
	if tok := p.l.Peek(); tok.Type == option {
		p.l.Next()
		return true
	}
	return false
}

func (p *parser) optionalv(option string) bool {
	if tok := p.l.Peek(); tok.Value == option {
		p.l.Next()
		return true
	}
	return false
}

func (p *parser) anythingBut(r rune) bool {
	return p.l.Peek().Type != r
}

func (p *parser) oneof(expectedTypes ...rune) Token {
	tok := p.l.Next()
	for _, t := range expectedTypes {
		if tok.Type == t {
			return tok
		}
	}
	p.fail(tok, "unexpected token %s, expected one of %s", tok.Value, strings.Join(reverseSymbols(expectedTypes), " "))
	return Token{}
}

func (p *parser) oneofval(expectedValues ...string) Token {
	tok := p.l.Next()
	for _, v := range expectedValues {
		if tok.Value == v {
			return tok
		}
	}
	p.fail(tok, "unexpected token %s, expected one of %s", tok.Value, strings.Join(expectedValues, ", "))
	return Token{}
}

func (p *parser) fail(pos Token, message string, args ...interface{}) {
	fail(p.l.filename, pos.Pos, message, args...)
}

func (p *parser) parseStatement() *Statement {
	s := &Statement{}
	tok := p.l.Peek()
	s.Pos = tok.Pos

	switch tok.Value {
	case "pass":
		s.Pass = true
		p.endPos = p.l.Next().EndPos()
		p.next(EOL)
	case "continue":
		s.Continue = true
		p.endPos = p.l.Next().EndPos()
		p.next(EOL)
	case "def":
		s.FuncDef = p.parseFuncDef()
	case "for":
		s.For = p.parseFor()
	case "if":
		s.If = p.parseIf()
	case "return":
		p.endPos = p.l.Next().EndPos()
		s.Return = p.parseReturn()
	case "raise":
		p.l.Next()
		s.Raise = p.parseExpression()
		p.next(EOL)
	case "assert":
		p.l.Next()
		s.Assert = &AssertStatement{
			Expr: p.parseExpression(),
		}
		if p.optional(',') {
			s.Assert.Message = p.parseExpression()
			p.endPos = s.Assert.Message.EndPos
		}
		p.next(EOL)
	default:
		if tok.Type == Ident {
			s.Ident = p.parseIdentStatement()
		} else {
			s.Literal = p.parseExpression()
		}
		p.next(EOL)
	}
	s.EndPos = p.endPos
	return s
}

func (p *parser) parseStatements() []*Statement {
	stmts := []*Statement{}
	for p.anythingBut(Unindent) {
		stmts = append(stmts, p.parseStatement())
	}
	p.next(Unindent)
	return stmts
}

func (p *parser) parseReturn() *ReturnStatement {
	r := &ReturnStatement{}
	for p.anythingBut(EOL) {
		r.Values = append(r.Values, p.parseExpression())
		if !p.optional(',') {
			break
		}
	}
	p.next(EOL)
	return r
}

func (p *parser) parseFuncDef() *FuncDef {
	p.nextv("def")
	fd := &FuncDef{
		Name: p.next(Ident).Value,
	}
	if strings.HasPrefix(fd.Name, "_") {
		fd.IsPrivate = true
	}
	p.next('(')
	for p.anythingBut(')') {
		fd.Arguments = append(fd.Arguments, p.parseArgument())
		if !p.optional(',') {
			break
		}
	}
	p.next(')')

	if tok := p.l.Peek(); tok.Value == "-" {
		p.next('-')
		p.next('>')

		tok := p.oneofval("bool", "str", "int", "list", "dict", "function", "config")
		fd.Return = tok.Value
	}

	// Get the position for the end of function defition header
	fd.EoDef = p.next(':').Pos

	p.next(EOL)
	if tok := p.l.Peek(); tok.Type == String {
		fd.Docstring = tok.Value
		// endPos being set here, this is for when function only contains docstring
		p.endPos = p.l.Next().EndPos()
		p.next(EOL)
	}

	fd.Statements = p.parseStatements()

	return fd
}

func (p *parser) parseArgument() Argument {
	a := Argument{
		Name: p.next(Ident).Value,
	}
	// indicate an argument is private if it is prefixed with "_"
	if strings.HasPrefix(a.Name, "_") {
		a.IsPrivate = true
	}
	if tok := p.l.Peek(); tok.Type == ',' || tok.Type == ')' {
		return a
	}
	tok := p.oneof(':', '&', '=')
	if tok.Type == ':' {
		// Type annotations
		for {
			tok = p.oneofval("bool", "str", "int", "list", "dict", "function", "config")
			a.Type = append(a.Type, tok.Value)
			if !p.optional('|') {
				break
			}
		}
		if tok := p.l.Peek(); tok.Type == ',' || tok.Type == ')' {
			return a
		}
		tok = p.oneof('&', '=')
	}
	if tok.Type == '&' {
		// Argument aliases
		for {
			tok = p.next(Ident)
			a.Aliases = append(a.Aliases, tok.Value)
			if !p.optional('&') {
				break
			}
		}
		if tok := p.l.Peek(); tok.Type == ',' || tok.Type == ')' {
			return a
		}
		p.next('=')
	}
	// Default value
	a.Value = p.parseExpression()
	return a
}

func (p *parser) parseIf() *IfStatement {
	p.nextv("if")
	i := &IfStatement{}
	p.parseExpressionInPlace(&i.Condition)
	p.next(':')
	p.next(EOL)
	i.Statements = p.parseStatements()

	for p.optionalv("elif") {
		elif := IfStatementElif{}
		p.parseExpressionInPlace(&elif.Condition)
		p.next(':')
		p.next(EOL)
		elif.Statements = p.parseStatements()
		i.Elif = append(i.Elif, elif)
	}
	if p.optionalv("else") {
		p.next(':')
		p.next(EOL)
		i.ElseStatements = p.parseStatements()
	}

	return i
}

func (p *parser) parseFor() *ForStatement {
	f := &ForStatement{}
	p.nextv("for")
	f.Names = p.parseIdentList()
	p.nextv("in")
	p.parseExpressionInPlace(&f.Expr)
	p.next(':')
	p.next(EOL)
	f.Statements = p.parseStatements()

	return f
}

// TODO: could ret last token here
func (p *parser) parseIdentList() []string {
	ret := []string{p.next(Ident).Value} // First one is compulsory
	for tok := p.l.Peek(); tok.Type == ','; tok = p.l.Peek() {
		p.l.Next()
		ret = append(ret, p.next(Ident).Value)
	}
	return ret
}

func (p *parser) parseExpression() *Expression {
	e := p.parseUnconditionalExpression()
	p.parseInlineIf(e)
	e.EndPos = p.endPos
	return e
}

func (p *parser) parseExpressionInPlace(e *Expression) {
	e.Pos = p.l.Peek().Pos
	p.parseUnconditionalExpressionInPlace(e)
	p.parseInlineIf(e)
	e.EndPos = p.endPos
}

func (p *parser) parseInlineIf(e *Expression) {
	if p.optionalv("if") {
		e.If = &InlineIf{Condition: p.parseExpression()}
		p.nextv("else")
		e.If.Else = p.parseExpression()
	}
}

func (p *parser) parseUnconditionalExpression() *Expression {
	e := &Expression{Pos: p.l.Peek().Pos}
	p.parseUnconditionalExpressionInPlace(e)
	return e
}

func (p *parser) parseUnconditionalExpressionInPlace(e *Expression) {
	if tok := p.l.Peek(); tok.Type == '-' || tok.Value == "not" {
		p.l.Next()
		e.UnaryOp = &UnaryOp{
			Op:   tok.Value,
			Expr: *p.parseValueExpression(),
		}
	} else {
		e.Val = p.parseValueExpression()
	}
	tok := p.l.Peek()
	if tok.Value == "and" || tok.Value == "or" {
		p.l.Next()
		lo := &LogicalOpExpression{Op: And}
		if tok.Value == "or" {
			lo.Op = Or
		}
		p.parseUnconditionalExpressionInPlace(&lo.Expr)
		e.Logical = lo
	} else {
		if tok.Value == "not" {
			// Hack for "not in" which needs an extra token.
			p.l.Next()
			tok = p.l.Peek()
			p.assert(tok.Value == "in", tok, "expected 'in', not %s", tok.Value)
			tok.Value = "not in"
			p.endPos = tok.EndPos()
		}
		if op, present := operators[tok.Value]; present {
			p.l.Next()
			o := OpExpression{Op: op}
			if op == Is {
				if tok := p.l.Peek(); tok.Value == "not" {
					// Mild hack for "is not" which needs to become a single operator.
					o.Op = IsNot
					p.endPos = tok.EndPos()
					p.l.Next()
				}
			}
			o.Expr = p.parseUnconditionalExpression()
			e.Op = append(e.Op, o)
			if len(o.Expr.Op) > 0 {
				e.Op = append(e.Op, o.Expr.Op...)
				o.Expr.Op = nil
			}
		}
	}
}

func concatStrings(lhs *ValueExpression, rhs *ValueExpression) *ValueExpression {
	// If they're both fStrngs
	if lhs.FString != nil && rhs.FString != nil {
		// If rhs has no variables, handle that
		if len(rhs.FString.Vars) == 0 {
			lhs.FString.Suffix += rhs.FString.Suffix
			return lhs
		}

		// Otherwise merge the vars
		rhs.FString.Vars[0].Prefix = lhs.FString.Suffix + rhs.FString.Vars[0].Prefix
		rhs.FString.Vars = append(lhs.FString.Vars, rhs.FString.Vars...)

		return rhs
	}

	// lhs is fString, add rhs to suffix
	if lhs.FString != nil && rhs.FString == nil {
		lhs.FString.Suffix += rhs.String[1 : len(rhs.String)-1]
		return lhs
	}

	// lhs is string, add rhs to prefix of first var
	if lhs.FString == nil && rhs.FString != nil {
		rhs.FString.Vars[0].Prefix = lhs.String[1:len(lhs.String)-1] + rhs.FString.Vars[0].Prefix
		return rhs
	}

	// otherwise they must both be strings
	lhs.String = "\"" + lhs.String[1:len(lhs.String)-1] + rhs.String[1:len(rhs.String)-1] + "\""
	return lhs
}

func (p *parser) parseValueExpression() *ValueExpression {
	ve := &ValueExpression{}
	tok := p.l.Peek()

	if tok.Type == String {
		if tok.Value[0] == 'f' {
			ve.FString = p.parseFString()
		} else {
			ve.String = tok.Value
			p.endPos = p.l.Next().EndPos()
		}

		if p.l.Peek().Type == String {
			return concatStrings(ve, p.parseValueExpression())
		}
	} else if tok.Type == Int {
		p.assert(len(tok.Value) < 19, tok, "int literal is too large: %s", tok)
		i, err := strconv.Atoi(tok.Value)
		p.assert(err == nil, tok, "invalid int value %s", tok) // Theoretically the lexer shouldn't have fed us this...
		ve.Int = i
		ve.IsInt = true
		p.endPos = p.l.Next().EndPos()
	} else if tok.Value == "False" {
		ve.False = true // hmmm...
		p.endPos = p.l.Next().EndPos()
	} else if tok.Value == "True" {
		ve.True = true
		p.endPos = p.l.Next().EndPos()
	} else if tok.Value == "None" {
		ve.None = true
		p.endPos = p.l.Next().EndPos()
	} else if tok.Type == '[' {
		ve.List = p.parseList('[', ']')
	} else if tok.Type == '(' {
		ve.Tuple = p.parseList('(', ')')
	} else if tok.Type == '{' {
		ve.Dict = p.parseDict()
	} else if tok.Value == "lambda" {
		ve.Lambda = p.parseLambda()
	} else if tok.Type == Ident {
		ve.Ident = p.parseIdentExpr()
		p.endPos = ve.Ident.EndPos
	} else {
		p.fail(tok, "Unexpected token %s", tok)
	}

	tok = p.l.Peek()
	for tok.Type == '[' {
		ve.Slices = append(ve.Slices, p.parseSlice())
		tok = p.l.Peek()
	}
	if p.optional('.') {
		ve.Property = p.parseIdentExpr()
		p.endPos = ve.Property.EndPos
	} else if p.optional('(') {
		ve.Call = p.parseCall()
	}
	return ve
}

func (p *parser) parseIdentStatement() *IdentStatement {
	tok := p.l.Peek()
	i := &IdentStatement{
		Name: p.next(Ident).Value,
	}
	_, reserved := keywords[i.Name]
	p.assert(!reserved, tok, "Cannot operate on keyword or constant %s", i.Name)
	if tok := p.l.Peek(); tok.Type == EOL {
		return i
	}
	tok = p.l.Next()
	switch tok.Type {
	case ',':
		i.Unpack = &IdentStatementUnpack{
			Names: p.parseIdentList(),
		}
		p.next('=')
		i.Unpack.Expr = p.parseExpression()
	case '[':
		i.Index = &IdentStatementIndex{
			Expr: p.parseExpression(),
		}
		p.endPos = p.next(']').EndPos()
		if tok := p.oneofval("=", "+="); tok.Type == '=' {
			i.Index.Assign = p.parseExpression()
		} else {
			i.Index.AugAssign = p.parseExpression()
		}
	case '.':
		i.Action = &IdentStatementAction{
			Property: p.parseIdentExpr(),
		}
		p.endPos = i.Action.Property.EndPos
	case '(':
		i.Action = &IdentStatementAction{
			Call: p.parseCall(),
		}
	case '=':
		i.Action = &IdentStatementAction{
			Assign: p.parseExpression(),
		}
	default:
		p.assert(tok.Value == "+=", tok, "Unexpected token %s, expected one of , [ . ( = +=", tok)
		i.Action = &IdentStatementAction{
			AugAssign: p.parseExpression(),
		}
	}
	return i
}

func (p *parser) parseIdentExpr() *IdentExpr {
	identTok := p.next(Ident)
	ie := &IdentExpr{
		Name: identTok.Value,
		Pos:  identTok.Pos,
	}
	for tok := p.l.Peek(); tok.Type == '.' || tok.Type == '('; tok = p.l.Peek() {
		tok := p.l.Next()
		action := IdentExprAction{}
		if tok.Type == '.' {
			action.Property = p.parseIdentExpr()
			ie.EndPos = action.Property.EndPos
		} else {
			action.Call = p.parseCall()
			ie.EndPos = p.endPos
		}
		ie.Action = append(ie.Action, action)
	}
	// In case the Ident is a variable name, we assign the endPos to the end of current token.
	// see test_data/unary_op.build
	if ie.EndPos == 0 {
		ie.EndPos = identTok.EndPos()
	}
	return ie
}

func (p *parser) parseCall() *Call {
	// The leading ( has already been consumed (because that fits better at the various call sites)
	c := &Call{}
	names := map[string]bool{}
	for tok := p.l.Peek(); tok.Type != ')'; tok = p.l.Peek() {
		arg := CallArgument{}
		if tok.Type == Ident && p.l.AssignFollows() {
			// Named argument.
			arg.Pos = tok.Pos
			arg.Name = tok.Value
			p.next(Ident)
			p.next('=')
			p.assert(!names[arg.Name], tok, "Repeated argument %s", arg.Name)
			names[arg.Name] = true
		}
		p.parseExpressionInPlace(&arg.Value)
		c.Arguments = append(c.Arguments, arg)
		if !p.optional(',') {
			break
		}
	}
	p.endPos = p.next(')').EndPos()
	return c
}

func (p *parser) parseList(opening, closing rune) *List {
	l := &List{}
	p.next(opening)
	for tok := p.l.Peek(); tok.Type != closing; tok = p.l.Peek() {
		l.Values = append(l.Values, p.parseExpression())
		if !p.optional(',') {
			break
		}
	}
	if tok := p.l.Peek(); tok.Value == "for" {
		p.assert(len(l.Values) == 1, tok, "Must have exactly 1 item in a list comprehension")
		l.Comprehension = p.parseComprehension()
	}
	p.endPos = p.next(closing).EndPos()
	return l
}

func (p *parser) parseDict() *Dict {
	d := &Dict{}
	p.next('{')
	for tok := p.l.Peek(); tok.Type != '}'; tok = p.l.Peek() {
		di := &DictItem{}
		p.parseExpressionInPlace(&di.Key)
		p.next(':')
		p.parseExpressionInPlace(&di.Value)
		d.Items = append(d.Items, di)
		if !p.optional(',') {
			break
		}
	}
	if tok := p.l.Peek(); tok.Value == "for" {
		p.assert(len(d.Items) == 1, tok, "Must have exactly 1 key:value pair in a dict comprehension")
		d.Comprehension = p.parseComprehension()
	}
	p.endPos = p.next('}').EndPos()
	return d
}

func (p *parser) parseSlice() *Slice {
	s := &Slice{}
	p.next('[')
	if p.optional(':') {
		s.Colon = ":"
	} else if !p.optional(':') {
		s.Start = p.parseExpression()
		if p.optional(':') {
			s.Colon = ":"
		}
	}
	if nextType := p.l.Peek().Type; nextType == ']' {
		p.endPos = p.l.Next().EndPos()
		return s
	}
	s.End = p.parseExpression()
	p.endPos = p.next(']').EndPos()
	return s
}

func (p *parser) parseComprehension() *Comprehension {
	c := &Comprehension{}
	p.nextv("for")
	c.Names = p.parseIdentList()
	p.nextv("in")
	c.Expr = p.parseUnconditionalExpression()
	if p.optionalv("for") {
		c.Second = &SecondComprehension{
			Names: p.parseIdentList(),
		}
		p.nextv("in")
		c.Second.Expr = p.parseUnconditionalExpression()
	}
	if p.optionalv("if") {
		c.If = p.parseUnconditionalExpression()
	}
	return c
}

func (p *parser) parseLambda() *Lambda {
	l := &Lambda{}
	p.nextv("lambda")
	for tok := p.l.Peek(); tok.Type == Ident; tok = p.l.Peek() {
		p.l.Next()
		arg := Argument{Name: tok.Value}
		if p.optional('=') {
			arg.Value = p.parseExpression()
		}
		l.Arguments = append(l.Arguments, arg)
		if !p.optional(',') {
			break
		}
	}
	p.next(':')
	p.parseExpressionInPlace(&l.Expr)
	return l
}

func (p *parser) parseFString() *FString {
	f := &FString{}
	tok := p.next(String)
	s := tok.Value[2 : len(tok.Value)-1] // Strip preceding f" and trailing "
	p.endPos = tok.EndPos()
	tok.Pos++ // track position in case of error
	for idx := p.findBrace(s); idx != -1; idx = p.findBrace(s) {
		v := FStringVar{
			Prefix: strings.ReplaceAll(strings.ReplaceAll(s[:idx], "{{", "{"), "}}", "}"),
		}
		s = s[idx+1:]
		tok.Pos += Position(idx + 1)
		idx = strings.IndexByte(s, '}')
		p.assert(idx != -1, tok, "Unterminated brace in fstring")
		v.Var = strings.Split(s[:idx], ".")
		f.Vars = append(f.Vars, v)
		s = s[idx+1:]
		tok.Pos += Position(idx + 1)
	}
	f.Suffix = strings.ReplaceAll(strings.ReplaceAll(s, "{{", "{"), "}}", "}")

	return f
}

func (p *parser) findBrace(s string) int {
	last := ' '
	for i, c := range s {
		if c == '{' && last != '{' && last != '$' {
			if i+1 < len(s) && s[i+1] == '{' {
				last = c
				continue
			}
			return i
		}
		last = c
	}
	return -1
}

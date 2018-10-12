package asp

import (
	"io"
	"reflect"
	"strconv"
	"strings"
)

// keywords are the list of reserved keywords in the language. They can't be assigned to.
// Not all of these have meaning in the build language (and many never will), but they are
// reserved in Python and for practical reasons it is useful to remain a subset of Python.
var keywords = map[string]struct{}{
	"False":    struct{}{},
	"None":     struct{}{},
	"True":     struct{}{},
	"and":      struct{}{},
	"as":       struct{}{},
	"assert":   struct{}{},
	"break":    struct{}{},
	"class":    struct{}{},
	"continue": struct{}{},
	"def":      struct{}{},
	"del":      struct{}{},
	"elif":     struct{}{},
	"else":     struct{}{},
	"except":   struct{}{},
	"finally":  struct{}{},
	"for":      struct{}{},
	"from":     struct{}{},
	"global":   struct{}{},
	"if":       struct{}{},
	"import":   struct{}{},
	"in":       struct{}{},
	"is":       struct{}{},
	"lambda":   struct{}{},
	"nonlocal": struct{}{},
	"not":      struct{}{},
	"or":       struct{}{},
	"pass":     struct{}{},
	"raise":    struct{}{},
	"return":   struct{}{},
	"try":      struct{}{},
	"while":    struct{}{},
	"with":     struct{}{},
	"yield":    struct{}{},
}

type parser struct {
	l *lex
}

// parseFileInput is the only external entry point to this class, it parses a file into a FileInput structure.
func parseFileInput(r io.Reader) (input *FileInput, err error) {
	// The rest of the parser functions signal unhappiness by panicking, we
	// recover any such failures here and convert to an error.
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	p := &parser{l: newLexer(r)}
	input = &FileInput{}
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
	fail(pos.Pos, message, args...)
}

func (p *parser) parseStatement() *Statement {
	s := &Statement{}
	tok := p.l.Peek()
	s.Pos = tok.Pos

	var endPos Position
	switch tok.Value {
	case "pass":
		s.Pass = true
		endPos = p.l.Next().EndPos()
		p.next(EOL)
	case "continue":
		s.Continue = true
		endPos = p.l.Next().EndPos()
		p.next(EOL)
	case "def":
		s.FuncDef, endPos = p.parseFuncDef()
	case "for":
		s.For, endPos = p.parseFor()
	case "if":
		s.If, endPos = p.parseIf()
	case "return":
		p.l.Next()
		s.Return, endPos = p.parseReturn()
	case "raise":
		p.l.Next()
		s.Raise = p.parseExpression()
		endPos = p.next(EOL).Pos
	case "assert":
		p.initField(&s.Assert)
		p.l.Next()
		s.Assert.Expr = p.parseExpression()
		if p.optional(',') {
			s.Assert.Message = p.next(String).Value
		}
		endPos = p.next(EOL).Pos
	default:
		if tok.Type == Ident {
			s.Ident, endPos = p.parseIdentStatement()
		} else {
			s.Literal = p.parseExpression()
			endPos = s.Literal.EndPos
		}
		nextTok := p.next(EOL)
		if endPos.Column == 0 {
			endPos = nextTok.Pos
		}
	}
	s.EndPos = endPos
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

func (p *parser) parseReturn() (*ReturnStatement, Position) {
	r := &ReturnStatement{}
	for p.anythingBut(EOL) {
		r.Values = append(r.Values, p.parseExpression())
		if !p.optional(',') {
			break
		}
	}
	endPos := p.next(EOL).EndPos()

	//TODO(bnm): This is a bit hacky, but I'm not sure if there is a better way :(
	if endPos.Column == 1 {
		endPos = r.Values[len(r.Values) - 1].Pos
	}
	return r, endPos
}

func (p *parser) parseFuncDef() (*FuncDef, Position) {
	p.nextv("def")
	fd := &FuncDef{
		Name: p.next(Ident).Value,
	}
	p.next('(')
	for p.anythingBut(')') {
		fd.Arguments = append(fd.Arguments, p.parseArgument())
		if !p.optional(',') {
			break
		}
	}
	p.next(')')
	// Get the position for the end of function defition header
	fd.EoDef = p.next(':').Pos

	p.next(EOL)
	var endPos Position
	if tok := p.l.Peek(); tok.Type == String {
		fd.Docstring = tok.Value
		// endPos being set here, this is for when function only contains docstring
		endPos = p.l.Next().EndPos()
		p.next(EOL)
	}

	fd.Statements = p.parseStatements()

	if len(fd.Statements) > 0 {
		endPos = fd.Statements[len(fd.Statements) - 1].EndPos
	}
	return fd, endPos
}

func (p *parser) parseArgument() Argument {
	a := Argument{
		Name: p.next(Ident).Value,
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
		tok = p.next('=')
	}
	// Default value
	a.Value = p.parseExpression()
	return a
}

func (p *parser) parseIf() (*IfStatement, Position) {
	p.nextv("if")
	i := &IfStatement{}
	p.parseExpressionInPlace(&i.Condition)
	p.next(':')
	p.next(EOL)
	i.Statements = p.parseStatements()

	allStatements := i.Statements
	for p.optionalv("elif") {
		elif := &i.Elif[p.newElement(&i.Elif)]
		p.parseExpressionInPlace(&elif.Condition)
		p.next(':')
		p.next(EOL)
		elif.Statements = p.parseStatements()
		allStatements = append(allStatements, elif.Statements...)
	}
	if p.optionalv("else") {
		p.next(':')
		p.next(EOL)
		i.ElseStatements = p.parseStatements()
		allStatements = append(allStatements, i.ElseStatements...)
	}

	endPos := allStatements[len(allStatements)-1].EndPos
	return i, endPos
}

// newElement is a nasty little hack to allow extending slices of types that we can't readily name.
// This is added in preference to having to break everything out to separately named types.
func (p *parser) newElement(x interface{}) int {
	v := reflect.ValueOf(x).Elem()
	v.Set(reflect.Append(v, reflect.Zero(v.Type().Elem())))
	return v.Len() - 1
}

// initField is a similar little hack for initialising non-slice fields.
func (p *parser) initField(x interface{}) {
	v := reflect.ValueOf(x).Elem()
	v.Set(reflect.New(v.Type().Elem()))
}

func (p *parser) parseFor() (*ForStatement, Position) {
	f := &ForStatement{}
	p.nextv("for")
	f.Names = p.parseIdentList()
	p.nextv("in")
	p.parseExpressionInPlace(&f.Expr)
	p.next(':')
	p.next(EOL)
	f.Statements = p.parseStatements()

	endPos := f.Statements[len(f.Statements) - 1].EndPos
	return f, endPos
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
	return e
}

func (p *parser) parseExpressionInPlace(e *Expression) {
	e.Pos = p.l.Peek().Pos
	p.parseUnconditionalExpressionInPlace(e)
	p.parseInlineIf(e)
}

func (p *parser) parseInlineIf(e *Expression) {
	if p.optionalv("if") {
		e.If = &InlineIf{Condition: p.parseExpression()}
		p.nextv("else")
		e.If.Else = p.parseExpression()
		e.EndPos = e.If.Else.EndPos
	}
}

func (p *parser) parseUnconditionalExpression() *Expression {
	e := &Expression{Pos: p.l.Peek().Pos}
	p.parseUnconditionalExpressionInPlace(e)
	return e
}

func (p *parser) parseUnconditionalExpressionInPlace(e *Expression) {
	var endPos Position
	if tok := p.l.Peek(); tok.Type == '-' || tok.Value == "not" {
		p.l.Next()
		var valueExp *ValueExpression
		valueExp, endPos = p.parseValueExpression()
		e.UnaryOp = &UnaryOp{
			Op:   tok.Value,
			Expr: *valueExp,
		}
	} else {
		e.Val, endPos = p.parseValueExpression()
	}
	tok := p.l.Peek()
	if tok.Value == "not" {
		// Hack for "not in" which needs an extra token.
		p.l.Next()
		tok = p.l.Peek()
		p.assert(tok.Value == "in", tok, "expected 'in', not %s", tok.Value)
		tok.Value = "not in"
		endPos = tok.EndPos()
	}
	if op, present := operators[tok.Value]; present {
		tok = p.l.Next()
		o := &e.Op[p.newElement(&e.Op)]
		o.Op = op
		o.Expr = p.parseUnconditionalExpression()
		if len(o.Expr.Op) > 0 {
			if op := o.Expr.Op[0].Op; op == And || op == Or || op == Is {
				// Hoist logical operator back up here to fix precedence. This is a bit of a hack and
				// might not be perfect in all cases...
				e.Op = append(e.Op, o.Expr.Op...)
				o.Expr.Op = nil
			}
		}
		p.l.Peek()
		endPos = o.Expr.EndPos
	}
	e.EndPos = endPos
}

func (p *parser) parseValueExpression() (*ValueExpression, Position) {
	ve := &ValueExpression{}
	tok := p.l.Peek()

	var endPos Position
	if tok.Type == String {
		if tok.Value[0] == 'f' {
			ve.FString = p.parseFString()
		} else {
			ve.String = tok.Value
			endPos = p.l.Next().EndPos()
		}
	} else if tok.Type == Int {
		p.assert(len(tok.Value) < 19, tok, "int literal is too large: %s", tok)
		p.initField(&ve.Int)
		i, err := strconv.Atoi(tok.Value)
		p.assert(err == nil, tok, "invalid int value %s", tok) // Theoretically the lexer shouldn't have fed us this...
		ve.Int.Int = i
		endPos = p.l.Next().EndPos()
	} else if tok.Value == "False" || tok.Value == "True" || tok.Value == "None" {
		ve.Bool = tok.Value
		endPos = p.l.Next().EndPos()
	} else if tok.Type == '[' {
		ve.List, endPos = p.parseList('[', ']')
	} else if tok.Type == '(' {
		ve.Tuple, endPos = p.parseList('(', ')')
	} else if tok.Type == '{' {
		ve.Dict, endPos = p.parseDict()
	} else if tok.Value == "lambda" {
		ve.Lambda = p.parseLambda()
	} else if tok.Type == Ident {
		ve.Ident, endPos = p.parseIdentExpr()

		// In case the Ident is a variable name, we assign the endPos to the end of current token.
		// see test_data/unary_op.build
		if endPos.Column == 0 {
			endPos = tok.EndPos()
		}
	} else {
		p.fail(tok, "Unexpected token %s", tok)
	}

	tok = p.l.Peek()
	if tok.Type == '[' {
		ve.Slice, endPos = p.parseSlice()
		tok = p.l.Peek()
	}
	if p.optional('.') {
		ve.Property, endPos = p.parseIdentExpr()
	} else if p.optional('(') {
		ve.Call, endPos = p.parseCall()
	}
	return ve, endPos
}

func (p *parser) parseIdentStatement() (*IdentStatement, Position) {
	tok := p.l.Peek()
	i := &IdentStatement{
		Name: p.next(Ident).Value,
	}
	_, reserved := keywords[i.Name]
	p.assert(!reserved, tok, "Cannot operate on keyword or constant %s", i.Name)
	tok = p.l.Next()
	var endPos Position
	switch tok.Type {
	case ',':
		p.initField(&i.Unpack)
		i.Unpack.Names = p.parseIdentList()
		p.next('=')
		i.Unpack.Expr = p.parseExpression()
		endPos = i.Unpack.Expr.EndPos
	case '[':
		p.initField(&i.Index)
		i.Index.Expr = p.parseExpression()
		endPos = p.next(']').EndPos()
		if tok := p.oneofval("=", "+="); tok.Type == '=' {
			i.Index.Assign = p.parseExpression()
			endPos = i.Index.Assign.EndPos
		} else {
			i.Index.AugAssign = p.parseExpression()
			endPos = i.Index.AugAssign.EndPos
		}
	case '.':
		p.initField(&i.Action)
		i.Action.Property, endPos = p.parseIdentExpr()
	case '(':
		p.initField(&i.Action)
		i.Action.Call, endPos = p.parseCall()
	case '=':
		p.initField(&i.Action)
		i.Action.Assign = p.parseExpression()
		endPos = i.Action.Assign.EndPos
	default:
		p.assert(tok.Value == "+=", tok, "Unexpected token %s, expected one of , [ . ( = +=", tok)
		p.initField(&i.Action)
		i.Action.AugAssign = p.parseExpression()
		endPos = i.Action.AugAssign.EndPos
	}
	return i, endPos
}

func (p *parser) parseIdentExpr() (*IdentExpr, Position) {
	var endPos Position
	ie := &IdentExpr{Name: p.next(Ident).Value}
	for tok := p.l.Peek(); tok.Type == '.' || tok.Type == '('; tok = p.l.Peek() {
		p.l.Next()
		action := &ie.Action[p.newElement(&ie.Action)]
		if tok.Type == '.' {
			action.Property, endPos = p.parseIdentExpr()
		} else {
			action.Call, endPos = p.parseCall()
		}
	}
	return ie, endPos
}

func (p *parser) parseCall() (*Call, Position) {
	// The leading ( has already been consumed (because that fits better at the various call sites)
	c := &Call{}
	names := map[string]bool{}
	for tok := p.l.Peek(); tok.Type != ')'; tok = p.l.Peek() {
		arg := CallArgument{}
		if tok.Type == Ident && p.l.AssignFollows() {
			// Named argument.
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
	endPos := p.next(')').EndPos()
	return c, endPos
}

func (p *parser) parseList(opening, closing rune) (*List, Position) {
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
	endPos := p.next(closing).EndPos()
	return l, endPos
}

func (p *parser) parseDict() (*Dict, Position) {
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
	endPos := p.next('}').EndPos()
	return d, endPos
}

func (p *parser) parseSlice() (*Slice, Position) {
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
		endPos := p.l.Next().EndPos()
		return s, endPos
	}
	s.End = p.parseExpression()
	endPos := p.next(']').EndPos()
	return s, endPos
}

func (p *parser) parseComprehension() *Comprehension {
	c := &Comprehension{}
	p.nextv("for")
	c.Names = p.parseIdentList()
	p.nextv("in")
	c.Expr = p.parseUnconditionalExpression()
	if p.optionalv("for") {
		p.initField(&c.Second)
		c.Second.Names = p.parseIdentList()
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
	tok.Pos.Column++                     // track position in case of error
	for idx := strings.IndexByte(s, '{'); idx != -1; idx = strings.IndexByte(s, '{') {
		v := &f.Vars[p.newElement(&f.Vars)]
		v.Prefix = s[:idx]
		s = s[idx+1:]
		tok.Pos.Column += idx + 1
		idx = strings.IndexByte(s, '}')
		p.assert(idx != -1, tok, "Unterminated brace in fstring")
		if varname := s[:idx]; strings.HasPrefix(varname, "CONFIG.") {
			v.Config = strings.TrimPrefix(varname, "CONFIG.")
		} else {
			v.Var = varname
		}
		s = s[idx+1:]
		tok.Pos.Column += idx + 1
	}
	f.Suffix = s
	return f
}

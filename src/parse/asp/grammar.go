package asp

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

// A FileInput is the top-level structure of a BUILD file.
type FileInput struct {
	Statements []*Statement `{ @@ } EOF`
}

// A Statement is the type we work with externally the most; it's a single Python statement.
// Note that some mildly excessive fiddling is needed since the parser we're using doesn't
// support backoff (i.e. if an earlier entry matches to its completion but can't consume
// following tokens, it doesn't then make another choice :( )
type Statement struct {
	Pos      lexer.Position
	Pass     string           `( @"pass" EOL`
	Continue string           `| @"continue" EOL`
	FuncDef  *FuncDef         `| @@`
	For      *ForStatement    `| @@`
	If       *IfStatement     `| @@`
	Return   *ReturnStatement `| "return" @@ EOL`
	Raise    *Expression      `| "raise" @@ EOL`
	Assert   *struct {
		Expr    *Expression `@@`
		Message string      `["," @String]`
	} `| "assert" @@ EOL`
	Ident   *IdentStatement `| @@ EOL`
	Literal *Expression     `| @@ EOL)`
}

// A ReturnStatement implements the Python 'return' statement.
type ReturnStatement struct {
	Values []*Expression `[ @@ { "," @@ } ]`
}

// A FuncDef implements definition of a new function.
type FuncDef struct {
	Name       string       `"def" @Ident`
	Arguments  []*Argument  `"(" [ @@ { "," @@ } ] ")" Colon EOL`
	Docstring  string       `[ @String EOL ]`
	Statements []*Statement `{ @@ } Unindent`
}

// A ForStatement implements the 'for' statement.
// Note that it does not support Python's "for-else" construction.
type ForStatement struct {
	Names      []string     `"for" @Ident [ { "," @Ident } ] "in"`
	Expr       Expression   `@@ Colon EOL`
	Statements []*Statement `{ @@ } Unindent`
}

// An IfStatement implements the if-elif-else statement.
type IfStatement struct {
	Condition  Expression   `"if" @@ Colon EOL`
	Statements []*Statement `{ @@ } Unindent`
	Elif       []struct {
		Condition  *Expression  `"elif" @@ Colon EOL`
		Statements []*Statement `{ @@ } Unindent`
	} `{ @@ }`
	ElseStatements []*Statement `[ "else" Colon EOL { @@ } Unindent ]`
}

// An Argument represents an argument to a function definition.
type Argument struct {
	Name string   `@Ident`
	Type []string `[ ":" @( { ( "bool" | "str" | "int" | "list" | "dict" | "function" ) [ "|" ] } ) ]`
	// Aliases are an experimental non-Python concept where function arguments can be aliased to different names.
	// We use this to support compatibility with Bazel & Buck etc in some cases.
	Aliases []string    `[ "&" ( { @Ident [ "&" ] } ) ]`
	Value   *Expression `[ "=" @@ ]`
}

// An Expression is a generalised Python expression, i.e. anything that can appear where an
// expression is allowed (including the extra parts like inline if-then-else, operators, etc).
type Expression struct {
	Pos     lexer.Position
	UnaryOp *UnaryOp         `( @@`
	Val     *ValueExpression `| @@ )`
	Op      *struct {
		Op   Operator    `@("+" | "-" | "%" | "<" | ">" | "and" | "or" | "is" | "in" | "not" "in" | "==" | "!=" | ">=" | "<=")`
		Expr *Expression `@@`
	} `[ @@ ]`
	If *InlineIf `[ @@ ]`
	// Not part of the grammar - applied later to optimise constant expressions.
	constant pyObject
	// Similarly applied to optimise simple lookups of local variables.
	local string
}

// A ValueExpression is the value part of an expression, i.e. without surrounding operators.
type ValueExpression struct {
	String string `( @String`
	Int    *struct {
		Int int `@Int`
	} `| @@` // Should just be *int, but https://github.com/golang/go/issues/23498 :(
	Bool     string     `| @( "True" | "False" | "None" )`
	List     *List      `| "[" @@ "]"`
	Dict     *Dict      `| "{" @@ "}"`
	Tuple    *List      `| "(" @@ ")"`
	Lambda   *Lambda    `| "lambda" @@`
	Ident    *IdentExpr `| @@ )`
	Slice    *Slice     `[ @@ ]`
	Property *IdentExpr `[ ( "." @@`
	Call     *Call      `| "(" @@ ")" ) ]`
}

// A UnaryOp represents a unary operation - in our case the only ones we support are negation and not.
type UnaryOp struct {
	Op   string          `@( "-" | "not" )`
	Expr ValueExpression `@@`
}

// An IdentStatement implements a statement that begins with an identifier (i.e. anything that
// starts off with a variable name). It is a little fiddly due to parser limitations.
type IdentStatement struct {
	Name   string `@Ident`
	Unpack *struct {
		Names []string    `@Ident { "," @Ident }`
		Expr  *Expression `"=" @@`
	} `( "," @@ `
	Index *struct {
		Expr      *Expression `@@ "]"`
		Assign    *Expression `( "=" @@`
		AugAssign *Expression `| "+=" @@ )`
	} `| "[" @@`
	Action *IdentStatementAction `| @@ )`
}

// An IdentStatementAction implements actions on an IdentStatement.
type IdentStatementAction struct {
	Property  *IdentExpr  `  "." @@`
	Call      *Call       `| "(" @@ ")"`
	Assign    *Expression `| "=" @@`
	AugAssign *Expression `| "+=" @@`
}

// An IdentExpr implements parts of an expression that begin with an identifier (i.e. anything
// that might be a variable name).
type IdentExpr struct {
	Name   string `@Ident`
	Action []struct {
		Property *IdentExpr `  "." @@`
		Call     *Call      `| "(" @@ ")"`
	} `{ @@ }`
}

// A Call represents a call site of a function.
type Call struct {
	Arguments []CallArgument `[ @@ ] { "," [ @@ ] }`
}

// A CallArgument represents a single argument at a call site of a function.
type CallArgument struct {
	Expr  *Expression `@@`
	Value *Expression `[ "=" @@ ]`
	self  pyObject    // Not part of the grammar, used later by interpreter for function calls.
}

// A List represents a list literal, either with or without a comprehension clause.
type List struct {
	Values        []*Expression  `[ @@ ] { "," [ @@ ] }`
	Comprehension *Comprehension `[ @@ ]`
}

// A Dict represents a dict literal, either with or without a comprehension clause.
type Dict struct {
	Items         []*DictItem    `[ @@ ] { "," [ @@ ] }`
	Comprehension *Comprehension `[ @@ ]`
}

// A DictItem represents a single key-value pair in a dict literal.
type DictItem struct {
	Key   string     `@( Ident | String ) ":"`
	Value Expression `@@`
}

// A Slice represents a slice or index expression (e.g. [1], [1:2], [2:], [:], etc).
type Slice struct {
	Start *Expression `"[" [ @@ ]`
	Colon string      `[ @":" ]`
	End   *Expression `[ @@ ] "]"`
}

// An InlineIf implements the single-line if-then-else construction
type InlineIf struct {
	Condition *Expression `"if" @@`
	Else      *Expression `[ "else" @@ ]`
}

// A Comprehension represents a list or dict comprehension clause.
type Comprehension struct {
	Names  []string    `"for" @Ident [ { "," @Ident } ] "in"`
	Expr   *Expression `@@`
	Second *struct {
		Names []string    `"for" @Ident [ { "," @Ident } ] "in"`
		Expr  *Expression `@@`
	} `[ @@ ]`
	If *Expression `[ "if" @@ ]`
}

// A Lambda is the inline lambda function.
type Lambda struct {
	Arguments []LambdaArgument `[ @@ { "," @@ } ] Colon`
	Expr      Expression       `@@`
}

// A LambdaArgument represents an argument to a lambda function.
// Vexingly these can't be normal function arguments any more, because the : for type annotations
// gets preferentially consumed to the one that ends the lambda itself :(
type LambdaArgument struct {
	Name  string      `@Ident`
	Value *Expression `[ "=" @@ ]`
}

// An Operator wraps up a Python binary operator to be faster to switch on
// and to add some useful extra methods.
type Operator int

const (
	// ComparisonOperator is used to mark comparison operators
	ComparisonOperator Operator = 0x100
	// LogicalOperator is used to mark logical operators.
	LogicalOperator Operator = 0x200
	// Add etc are arithmetic operators - these are implemented on a per-type basis
	Add Operator = iota
	// Subtract implements binary - (only works on integers)
	Subtract
	// Modulo implements % (including string interpolation)
	Modulo
	// LessThan implements <
	LessThan
	// GreaterThan implements >
	GreaterThan
	// LessThanOrEqual implements <=
	LessThanOrEqual
	// GreaterThanOrEqual implements >=
	GreaterThanOrEqual
	// Equal etc are comparison operators - also on a per-type basis but have slightly different rules.
	Equal = iota | ComparisonOperator
	// NotEqual implements !=
	NotEqual
	// In implements the in operator
	In
	// NotIn implements "not in" as a single operator.
	NotIn
	// And etc are logical operators - these are implemented type-independently
	And Operator = iota | LogicalOperator
	// Or implements the or operator
	Or
	// Is implements type identity.
	Is
	// Index is used in the parser, but not when parsing code.
	Index = -iota
)

// Capture implements capturing for the parser.
func (o *Operator) Capture(values []string) error {
	op, present := operators[strings.Join(values, " ")]
	if !present {
		return fmt.Errorf("Unknown operator: %s", values[0])
	}
	*o = op
	return nil
}

// IsComparison returns true if this operator is a comparison operator.
func (o Operator) IsComparison() bool {
	return (o & ComparisonOperator) == ComparisonOperator
}

// IsLogical returns true if this operator is a logical operator.
func (o Operator) IsLogical() bool {
	return (o & LogicalOperator) == LogicalOperator
}

// String implements the fmt.Stringer interface. It is not especially efficient and is
// normally only used for errors & debugging.
func (o Operator) String() string {
	for k, v := range operators {
		if o == v {
			return k
		}
	}
	return "unknown"
}

var operators = map[string]Operator{
	"+":      Add,
	"-":      Subtract,
	"%":      Modulo,
	"<":      LessThan,
	">":      GreaterThan,
	"and":    And,
	"or":     Or,
	"is":     Is,
	"in":     In,
	"not in": NotIn,
	"==":     Equal,
	"!=":     NotEqual,
	">=":     GreaterThanOrEqual,
	"<=":     LessThanOrEqual,
}

package asp

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/lexer"
)

// A fileInput is the top-level structure of a BUILD file.
type fileInput struct {
	Statements []*statement `{ @@ } EOF`
}

// A statement is the type we work with externally the most; it's a single Python statement.
// Note that some mildly excessive fiddling is needed since the parser we're using doesn't
// support backoff (i.e. if an earlier entry matches to its completion but can't consume
// following tokens, it doesn't then make another choice :( )
type statement struct {
	Pos      lexer.Position
	Pass     string           `( @"pass" EOL`
	Continue string           `| @"continue" EOL`
	FuncDef  *funcDef         `| @@`
	For      *forStatement    `| @@`
	If       *ifStatement     `| @@`
	Return   *returnStatement `| "return" @@ EOL`
	Raise    *expression      `| "raise" @@ EOL`
	Assert   *struct {
		Expr    *expression `@@`
		Message string      `["," @String]`
	} `| "assert" @@ EOL`
	Ident   *identStatement `| @@ EOL`
	Literal *expression     `| @@ EOL)`
}

type returnStatement struct {
	Values []*expression `[ @@ { "," @@ } ]`
}

type funcDef struct {
	Name string `"def" @Ident`
	// *args and **kwargs are not properly supported but we allow them here for some level of compatibility.
	Arguments  []*argument  `"(" [ @@ { "," { "*" } @@ } ] ")" Colon EOL`
	Docstring  string       `[ @String EOL ]`
	Statements []*statement `{ @@ } Unindent`
}

type forStatement struct {
	Names      []string     `"for" @Ident [ { "," @Ident } ] "in"`
	Expr       expression   `@@ Colon EOL`
	Statements []*statement `{ @@ } Unindent`
}

type ifStatement struct {
	Condition  expression   `"if" @@ Colon EOL`
	Statements []*statement `{ @@ } Unindent`
	Elif       []struct {
		Condition  *expression  `"elif" @@ Colon EOL`
		Statements []*statement `{ @@ } Unindent`
	} `{ @@ }`
	ElseStatements []*statement `[ "else" Colon EOL { @@ } Unindent ]`
}

type argument struct {
	Name  string      `@Ident`
	Type  []string    `[ ":" @( { ( "bool" | "str" | "int" | "list" | "dict" | "function" ) [ "|" ] } ) ]`
	Value *expression `[ "=" @@ ]`
}

type expression struct {
	Pos      lexer.Position
	UnaryOp  *unaryOp  `( @@`
	String   string    `| @String`
	Int      *intl     `| @@` // Should just be *int, but https://github.com/golang/go/issues/23498 :(
	Bool     string    `| @( "True" | "False" | "None" )`
	List     *list     `| "[" @@ "]"`
	Dict     *dict     `| "{" @@ "}"`
	Tuple    *list     `| "(" @@ ")"`
	Lambda   *lambda   `| "lambda" @@`
	Ident    *ident    `| @@ )`
	Slice    *slice    `[ @@ ]`
	Property *ident    `[ ( "." @@`
	Call     *call     `| "(" @@ ")" ) ]`
	Op       *operator `[ @@ ]`
	If       *inlineIf `[ @@ ]`
	// Not part of the grammar - applied later to optimise constant expressions.
	Constant pyObject
	// Similarly applied to optimise simple lookups of local variables.
	Local string
}

type intl struct {
	Int int `@Int`
}

type unaryOp struct {
	Op   string     `@( "-" | "not" )`
	Expr expression `@@`
}

type identStatement struct {
	Name   string `@Ident`
	Unpack *struct {
		Names []string    `@Ident { "," @Ident }`
		Expr  *expression `"=" @@`
	} `( "," @@ `
	Index *struct {
		Expr      *expression `@@ "]"`
		Assign    *expression `( "=" @@`
		AugAssign *expression `| "+=" @@ )`
	} `| "[" @@`
	Action *identStatementAction `| @@ )`
}

type identStatementAction struct {
	Property  *ident      `  "." @@`
	Call      *call       `| "(" @@ ")"`
	Assign    *expression `| "=" @@`
	AugAssign *expression `| "+=" @@`
}

type ident struct {
	Name   string `@Ident`
	Action []struct {
		Property *ident `  "." @@`
		Call     *call  `| "(" @@ ")"`
	} `{ @@ }`
}

type call struct {
	Arguments []callArgument `[ @@ ] { "," [ @@ ] }`
}

type callArgument struct {
	Expr  *expression `@@`
	Value *expression `[ "=" @@ ]`
	Self  pyObject    // Not part of the grammar, used later by interpreter for function calls.
}

type list struct {
	Values        []*expression  `[ @@ ] { "," [ @@ ] }`
	Comprehension *comprehension `[ @@ ]`
}

type dict struct {
	Items         []*dictItem    `[ @@ ] { "," [ @@ ] }`
	Comprehension *comprehension `[ @@ ]`
}

type dictItem struct {
	Key   string     `@( Ident | String ) ":"`
	Value expression `@@`
}

type operator struct {
	Op   Operator    `@("+" | "-" | "%" | "<" | ">" | "and" | "or" | "is" | "in" | "not" "in" | "==" | "!=" | ">=" | "<=")`
	Expr *expression `@@`
}

type slice struct {
	// Implements indexing as well as slicing.
	Start *expression `"[" [ @@ ]`
	Colon string      `[ @":" ]`
	End   *expression `[ @@ ] "]"`
}

type inlineIf struct {
	Condition *expression `"if" @@`
	Else      *expression `[ "else" @@ ]`
}

type comprehension struct {
	Names []string    `"for" @Ident [ { "," @Ident } ] "in"`
	Expr  *expression `@@`
	If    *expression `[ "if" @@ ]`
}

type lambda struct {
	Arguments []lambdaArgument `[ @@ { "," @@ } ] Colon`
	Expr      expression       `@@`
}

// Vexingly these can't be normal function arguments any more, because the : for type annotations
// gets preferentially consumed to the one that ends the lambda itself :(
type lambdaArgument struct {
	Name  string      `@Ident`
	Value *expression `[ "=" @@ ]`
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

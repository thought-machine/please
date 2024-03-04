package asp

// A FileInput is the top-level structure of a BUILD file.
type FileInput struct {
	Statements []*Statement
}

// A Position describes a position in a source file. It is opaque from the outside
// (but is implemented as simply a byte offset into the file).
// Most usages should begin by converting it to a FilePosition which is generally more useful;
// this form is optimised for size within the AST structures.
type Position int

// A Statement is the type we work with externally the most; it's a single Python statement.
// Note that some mildly excessive fiddling is needed since the parser we're using doesn't
// support backoff (i.e. if an earlier entry matches to its completion but can't consume
// following tokens, it doesn't then make another choice :( )
type Statement struct {
	Pos      Position
	EndPos   Position
	FuncDef  *FuncDef
	For      *ForStatement
	If       *IfStatement
	Return   *ReturnStatement
	Raise    *Expression // Deprecated
	Assert   *AssertStatement
	Ident    *IdentStatement
	Literal  *Expression
	Pass     bool
	Continue bool
}

// An AssertStatement implements the 'assert' statement.
type AssertStatement struct {
	Expr    *Expression
	Message *Expression
}

// A ReturnStatement implements the Python 'return' statement.
type ReturnStatement struct {
	Values []*Expression
}

// A FuncDef implements definition of a new function.
type FuncDef struct {
	Name       string
	Arguments  []Argument
	Docstring  string
	Statements []*Statement
	EoDef      Position
	// allowed return type of the FuncDef
	Return string
	// Not part of the grammar. Used to indicate internal targets that can only
	// be called using keyword arguments.
	KeywordsOnly bool
	// Indicates whether the function is private, i.e. name starts with an underscore.
	IsPrivate bool
	// True if the function is builtin to Please.
	IsBuiltin bool
}

// A ForStatement implements the 'for' statement.
// Note that it does not support Python's "for-else" construction.
type ForStatement struct {
	Names      []string
	Expr       Expression
	Statements []*Statement
}

// An IfStatement implements the if-elif-else statement.
type IfStatement struct {
	Condition      Expression
	Statements     []*Statement
	Elif           []IfStatementElif
	ElseStatements []*Statement
}

// An IfStatementElif holds an elif clause in the if-elif-else statement.
type IfStatementElif struct {
	Condition  Expression
	Statements []*Statement
}

// An Argument represents an argument to a function definition.
type Argument struct {
	Name string
	Type []string
	// Aliases are an experimental non-Python concept where function arguments can be aliased to different names.
	// We use this to support compatibility with Bazel & Buck etc in some cases.
	Aliases []string
	Value   *Expression

	IsPrivate bool
}

// An Expression is a generalised Python expression, i.e. anything that can appear where an
// expression is allowed (including the extra parts like inline if-then-else, operators, etc).
type Expression struct {
	Pos       Position
	EndPos    Position
	Val       *ValueExpression
	Op        []OpExpression
	If        *InlineIf
	optimised *optimisedExpression
}

// An optimisedExpression contains information to optimise certain aspects of execution of an expression.
type optimisedExpression struct {
	// Used to optimise constant expressions.
	Constant pyObject
	// Similarly applied to optimise simple lookups of local variables.
	Local string
	// And similarly applied to optimise lookups into configuration.
	Config string
	// String join expressions (like ' '.join([...])) where we avoid reifying the internal list.
	Join *optimisedJoin
}

type optimisedJoin struct {
	Base string
	List *List
}

// An OpExpression is a operator combined with its following expression.
type OpExpression struct {
	Op   Operator
	Expr *Expression
}

// A ValueExpression is the value part of an expression, i.e. without surrounding operators.
type ValueExpression struct {
	String  string
	FString *FString
	// These are true if this represents one of the boolean singletons
	True  bool
	False bool
	None  bool
	// True if the Int field is set; this helps us distinguish values of 0.
	IsInt    bool
	Int      int
	List     *List
	Dict     *Dict
	Tuple    *List
	Lambda   *Lambda
	Ident    *IdentExpr
	Slices   []*Slice
	Property *IdentExpr
	Call     *Call
}

// An FString represents a minimal version of a Python literal format string.
// Note that we only support a very small subset of what Python allows there; essentially only
// variable substitution, which gives a much simpler AST structure here.
type FString struct {
	Vars   []FStringVar
	Suffix string // Following string bit
}

// An FStringVar represents a single variable in an FString.
type FStringVar struct {
	Prefix string   // Preceding string bit
	Var    []string // Variable name to interpolate, plus any accessors
}

// An IdentStatement implements a statement that begins with an identifier (i.e. anything that
// starts off with a variable name). It is a little fiddly due to parser limitations.
type IdentStatement struct {
	Name   string
	Unpack *IdentStatementUnpack
	Index  *IdentStatementIndex
	Action *IdentStatementAction
}

// An IdentStatementUnpack implements unpacking on an IdentStatement.
type IdentStatementUnpack struct {
	Names []string
	Expr  *Expression
}

// An IdentStatementIndex implements indexing on an IdentStatement.
type IdentStatementIndex struct {
	Expr      *Expression
	Assign    *Expression
	AugAssign *Expression
}

// An IdentStatementAction implements actions on an IdentStatement.
type IdentStatementAction struct {
	Property  *IdentExpr
	Call      *Call
	Assign    *Expression
	AugAssign *Expression
}

// An IdentExpr implements parts of an expression that begin with an identifier (i.e. anything
// that might be a variable name).
type IdentExpr struct {
	Pos    Position
	EndPos Position
	Name   string
	Action []IdentExprAction
}

// An IdentExprAction represents an Action within an IdentExpr.
type IdentExprAction struct {
	Property *IdentExpr
	Call     *Call
}

// A Call represents a call site of a function.
type Call struct {
	Arguments []CallArgument
}

// A CallArgument represents a single argument at a call site of a function.
type CallArgument struct {
	Pos   Position
	Name  string
	Value Expression
}

// A List represents a list literal, either with or without a comprehension clause.
type List struct {
	Values        []*Expression
	Comprehension *Comprehension
}

// A Dict represents a dict literal, either with or without a comprehension clause.
type Dict struct {
	Items         []*DictItem
	Comprehension *Comprehension
}

// A DictItem represents a single key-value pair in a dict literal.
type DictItem struct {
	Key   Expression
	Value Expression
}

// A Slice represents a slice or index expression (e.g. [1], [1:2], [2:], [:], etc).
type Slice struct {
	Start *Expression
	Colon string
	End   *Expression
}

// An InlineIf implements the single-line if-then-else construction
type InlineIf struct {
	Condition *Expression
	Else      *Expression
}

// A Comprehension represents a list or dict comprehension clause.
type Comprehension struct {
	Names  []string
	Expr   *Expression
	Second *SecondComprehension
	If     *Expression
}

// A SecondComprehension represents a second 'for' clause in a list or dict comprehension.
type SecondComprehension struct {
	Names []string
	Expr  *Expression
}

// A Lambda is the inline lambda function.
type Lambda struct {
	Arguments []Argument
	Expr      Expression
}

// An Operator defines a unary or binary operator.
type Operator rune

const (
	// Add etc are arithmetic operators - these are implemented on a per-type basis
	Add Operator = '+'
	// Subtract implements binary - (only works on integers)
	Subtract Operator = '-'
	// Multiply implements multiplication between two types
	Multiply Operator = '×'
	// Divide implements division, currently only between integers
	Divide Operator = '÷'
	// Modulo implements % (including string interpolation)
	Modulo Operator = '%'
	// Negate is the unary negation operator (not exactly the same as Subtract)
	Negate Operator = '−'
	// LessThan implements <
	LessThan Operator = '<'
	// GreaterThan implements >
	GreaterThan Operator = '>'
	// LessThanOrEqual implements <=
	LessThanOrEqual Operator = '≤'
	// GreaterThanOrEqual implements >=
	GreaterThanOrEqual Operator = '≥'
	// Equal etc are comparison operators - also on a per-type basis but have slightly different rules.
	Equal Operator = '＝'
	// NotEqual implements !=
	NotEqual Operator = '≠'
	// In implements the in operator
	In Operator = '∈'
	// NotIn implements "not in" as a single operator.
	NotIn Operator = '∉'
	// And etc are logical operators - these are implemented type-independently
	And Operator = '&'
	// Or implements the or operator
	Or Operator = '∨'
	// Not implements the logical not operator (distinct from 'not in' or 'is not')
	Not Operator = '!'
	// Union implements the | or binary or operator, which is only used for dict unions.
	Union Operator = '∪'
	// Is implements type identity.
	Is Operator = '≡'
	// IsNot is the inverse of Is.
	IsNot Operator = '≢'
	// Index is used in the parser, but not when parsing code.
	Index Operator = '['
)

// String implements the fmt.Stringer interface. It is not especially efficient and is
// normally only used for errors & debugging.
func (o Operator) String() string {
	for k, v := range operators {
		if o == v {
			return k
		}
	}
	if o == Negate {
		return "-" // Can't have both this and Subtract in the map
	}
	return "unknown"
}

// Precedence returns the precedence of this operator (higher number == more tightly binding)
// The value has no particular meaning other than to compare
func (o Operator) Precedence() int {
	switch o {
	case Negate:
		return 4
	case Multiply, Divide, Modulo:
		return 3
	case Add, Subtract:
		return 2
	case Union:
		return 1
	case Not:
		return -1
	case And:
		return -2
	case Or:
		return -3
	default:
		return 0
	}
}

// Lazy returns true if the operand of this operator should be lazily evaluated (e.g. and, or)
func (o Operator) Lazy() bool {
	return o == And || o == Or
}

var operators = map[string]Operator{
	"+":      Add,
	"-":      Subtract,
	"*":      Multiply,
	"/":      Divide,
	"%":      Modulo,
	"<":      LessThan,
	">":      GreaterThan,
	"and":    And,
	"or":     Or,
	"is":     Is,
	"is not": IsNot,
	"in":     In,
	"not in": NotIn,
	"==":     Equal,
	"!=":     NotEqual,
	">=":     GreaterThanOrEqual,
	"<=":     LessThanOrEqual,
	"|":      Union,
	"not":    Not,
}

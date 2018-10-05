package lsp

// EOL is a list of options for end of line characters
var EOL = []string{"\n", "\r\n", "\r"}

// DocumentURI is the uri representation of the filepath, usually prefixed with "files://"
type DocumentURI string

// RequestCancelled is an error code specific to language server protocol
// it is been used when the requests returns an error response on cancellation
const RequestCancelled	int64 = -32800

// Position is the position in a text document expressed as zero-based line and zero-based character offset
type Position struct {
	/**
	 * Line position in a document (zero-based).
	 */
	Line int `json:"line"`

	/**
	 * Character offset on a line in a document (zero-based).
	 */
	Character int `json:"character"`
}

// Range is A range in a text document expressed as (zero-based) start and end positions.
// A range is comparable to a selection in an editor
type Range struct {
	/**
	 * The range's start position.
	 */
	Start Position `json:"start"`

	/**
	 * The range's end position.
	 */
	End Position `json:"end"`
}

// Location represents a location inside a resource, such as a line inside a text file.
type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

// Diagnostic represents a diagnostic, such as a compiler error or warning.
// Diagnostic objects are only valid in the scope of a resource.
type Diagnostic struct {
	/**
	 * The range at which the message applies.
	 */
	Range Range `json:"range"`

	/**
	 * The diagnostic's severity. Can be omitted. If omitted it is up to the
	 * client to interpret diagnostics as error, warning, info or hint.
	 */
	Severity DiagnosticSeverity `json:"severity,omitempty"`

	/**
	 * The diagnostic's code. Can be omitted.
	 */
	Code string `json:"code,omitempty"`

	/**
	 * A human-readable string describing the source of this
	 * diagnostic, e.g. 'typescript' or 'super lint'.
	 */
	Source string `json:"source,omitempty"`

	/**
	 * The diagnostic's message.
	 */
	Message string `json:"message"`

	/**
	 * An array of related diagnostic information, e.g. when symbol-names within
	 * a scope collide all definitions can be marked via this property.
	 */
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation"`
}

// DiagnosticRelatedInformation represents a related message and source code location for a diagnostic. This should be
// used to point to code locations that cause or related to a diagnostics, e.g when duplicating
// a symbol in a scope.
type DiagnosticRelatedInformation struct {
	/**
	 * The location of this related diagnostic information.
	 */
	Location Location `json:"location"`

	/**
	 * The message of this related diagnostic information.
	 */
	Message string `json:"message"`
}


// Command Represents a reference to a command.
// Provides a title which will be used to represent a command in the UI.
// Commands are identified by a string identifier.
type Command struct {
	/**
	 * Title of the command, like `save`.
	 */
	Title string `json:"title"`

	/**
	 * The identifier of the actual command handler.
	 */
	Command string `json:"command"`

	/**
	 * Arguments that the command handler should be
	 * invoked with.
	 */
	Arguments []interface{} `json:"arguments"`
}

// MarkedString can be used to render human readable text.
// TODO(bnmetrics): this might not be needed anymore
type MarkedString struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

// MarkupContent represents a string value which content can be represented in different formats.
type MarkupContent struct {
	Kind 	MarkupKind  `json:"kind"`
	Value 	string 		`json:"value"`
}

// Describes the content type that a client supports in various result literals
// like `Hover`, `ParameterInfo` or `CompletionItem`.
// `MarkupKinds` must not start with a `$`
type MarkupKind string

// Two types of MarkupKind
const (
	PlainText MarkupKind = "plaintext"
	MarkDown  MarkupKind = "markdown"
)

// DocumentFilter denotes a document through properties like language, scheme or pattern.
// TODO: not sure this is useful...As I think this has to do with specific languages on the list
type DocumentFilter struct {
}

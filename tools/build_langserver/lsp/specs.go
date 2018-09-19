package lsp

var EOL = []string{"\n", "\r\n", "\r"}

type DocumentURI string

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

type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

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

//type DiagnosticSeverity map[string]int
type DiagnosticSeverity int

const (
	Error       DiagnosticSeverity = 1
	Warning     DiagnosticSeverity = 2
	Information DiagnosticSeverity = 3
	Hint        DiagnosticSeverity = 4
)

/**
 * Represents a related message and source code location for a diagnostic. This should be
 * used to point to code locations that cause or related to a diagnostics, e.g when duplicating
 * a symbol in a scope.
 */
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


// TODO: not sure this is useful...As I think this has to do with specific languages on the list
type DocumentFilter struct {
}

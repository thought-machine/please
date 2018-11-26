package lsp

// CompletionItemKind indicate a completion entry
type CompletionItemKind int

// List of CompletionItemKind mapped to respective integer values
const (
	Text      CompletionItemKind = 1
	Method    CompletionItemKind = 2
	Function  CompletionItemKind = 3
	Field     CompletionItemKind = 4
	Variable  CompletionItemKind = 6
	Module    CompletionItemKind = 9
	Property  CompletionItemKind = 10
	Unit      CompletionItemKind = 11
	Value     CompletionItemKind = 12
	Keyword   CompletionItemKind = 14
	File      CompletionItemKind = 17
	Reference CompletionItemKind = 18
	Folder    CompletionItemKind = 19
	Operator  CompletionItemKind = 24
)

// DiagnosticSeverity type is for different kind of supported diagnostic severities
type DiagnosticSeverity int

// List of DiagnosticSeverity mapped to respective integer values
const (
	Error       DiagnosticSeverity = 1
	Warning     DiagnosticSeverity = 2
	Information DiagnosticSeverity = 3
	Hint        DiagnosticSeverity = 4
)

// TextDocumentSyncKind defines how the host (editor) should sync document changes to the language server.
type TextDocumentSyncKind int

// List of TextDocumentSyncKind mapped to respective integer values
const (
	SyncNone        TextDocumentSyncKind = 0
	SyncFull        TextDocumentSyncKind = 1
	SyncIncremental TextDocumentSyncKind = 2
)

// CompletionTriggerKind defines how a completion was triggered
type CompletionTriggerKind int

const (
	// Invoked means Completion was triggered by typing an identifier (24x7 code
	// complete), manual invocation (e.g Ctrl+Space) or via API.
	Invoked CompletionItemKind = 1

	// TriggerCharacter means Completion was triggered by a trigger character specified by
	//the `triggerCharacters` properties of the `CompletionRegistrationOptions`.
	TriggerCharacter CompletionItemKind = 2

	// TriggerForIncompleteCompletions was re-triggered as the current
	// completion list is incomplete.
	TriggerForIncompleteCompletions CompletionItemKind = 3
)

// InsertTextFormat defines whether the insert text in a completion item should be interpreted as
// plain text or a snippet.
type InsertTextFormat int

const (
	// ITFPlainText The primary text to be inserted is treated as a plain string.
	ITFPlainText InsertTextFormat = 1
	// ITFSnippet is the primary text to be inserted is treated as a snippet.
	ITFSnippet InsertTextFormat = 2
)

// TextDocumentSaveReason Represents reasons why a text document is saved.
type TextDocumentSaveReason int

// Represents reasons why a text document is saved.
const (
	Manual     TextDocumentSaveReason = 1
	AfterDelay TextDocumentSaveReason = 2
	FocusOut   TextDocumentSaveReason = 3
)

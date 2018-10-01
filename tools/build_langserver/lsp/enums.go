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

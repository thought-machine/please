package lsp


type CompletionItemKind int

const (
	Text 		CompletionItemKind = 1
	Method 		CompletionItemKind = 2
	Function	CompletionItemKind = 3
	Field		CompletionItemKind = 4
	Variable	CompletionItemKind = 6
	Module 		CompletionItemKind = 9
	Property	CompletionItemKind = 10
	Unit		CompletionItemKind = 11
	Value 		CompletionItemKind = 12
	Keyword 	CompletionItemKind = 14
	File		CompletionItemKind = 17
	Reference 	CompletionItemKind = 18
	Folder 		CompletionItemKind = 19
	Operator	CompletionItemKind = 24

)


type DiagnosticSeverity int

const (
	Error       DiagnosticSeverity = 1
	Warning     DiagnosticSeverity = 2
	Information DiagnosticSeverity = 3
	Hint        DiagnosticSeverity = 4
)

type TextDocumentSyncKind int

const (
	SyncNone        TextDocumentSyncKind = 0
	SyncFull        TextDocumentSyncKind = 1
	SyncIncremental TextDocumentSyncKind = 2
)

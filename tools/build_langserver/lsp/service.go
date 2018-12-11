package lsp

import "github.com/sourcegraph/jsonrpc2"

// ServerCapabilities Defines how text documents are synced.
type ServerCapabilities struct {
	/**
	 * Defines how text documents are synced. Is either a detailed structure defining each notification or
	 * for backwards compatibility the TextDocumentSyncKind number. If omitted it defaults to `TextDocumentSyncKind.None`.
	 */
	TextDocumentSync *TextDocumentSyncKind `json:"textDocumentSync,omitempty"`

	HoverProvider              bool                  `json:"hoverProvider"`
	CompletionProvider         *CompletionOptions    `json:"completionProvider"`
	SignatureHelpProvider      *SignatureHelpOptions `json:"signatureHelpProvider"`
	DefinitionProvider         bool                  `json:"definitionProvider"`
	TypeDefinitionProvider     bool                  `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider     bool                  `json:"implementationProvider,omitempty"`
	ReferencesProvider         bool                  `json:"referenceProvider,omitempty"`
	RenameProvider             bool                  `json:"renameProvider,omitempty"`
	DocumentSymbolProvider     bool                  `json:"documentSymbolProvider,omitempty"`
	DocumentHighlightProvider  bool                  `json:"documentHighlightProvider,omitempty"`
	DocumentFormattingProvider bool                  `json:"documentFormattingProvider,omitempty"`
}

// TextDocumentPositionParams is a parameter literal used in requests to pass a text document
// and a position inside that document
type TextDocumentPositionParams struct {
	/**
	 * The text document.
	 */
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	/**
	 * The position inside the text document.
	 */
	Position Position `json:"position"`
}

// TextDocumentSyncOptions defines the open and close notifications are sent to the server.
// TODO(bnmetrics): this might not be needed
type TextDocumentSyncOptions struct {
	OpenClose        bool                  `json:"openClose"`
	Change           *TextDocumentSyncKind `json:"change"`
	WillSave         bool                  `json:"willSave,omitempty"`
	WillSaveWaitUtil bool                  `json:"willSaveWaitUntil"`
	Save             *SaveOptions          `json:"save"`
}

// SaveOptions are the options for dealing with saving files
type SaveOptions struct {
	/**
	 * The client is supposed to include the content on save.
	 */
	IncludeText bool `json:"includeText"`
}

// CompletionOptions is a list of options server provides for completion support
type CompletionOptions struct {
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// SignatureHelpOptions indicate the server provides signature help support
type SignatureHelpOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// SignatureHelp is the response from "textDocument/signatureHelp"
// represents the signature of something callable.
type SignatureHelp struct {
	Signatures      []SignatureInformation `json:"signatures"`
	ActiveSignature int                    `json:"activeSignature"`
	ActiveParameter int                    `json:"activeParameter"`
}

// SignatureInformation represents the signature of something callable.
type SignatureInformation struct {
	Label         string                 `json:"label"`
	Documentation string                 `json:"documentation,omitempty"`
	Parameters    []ParameterInformation `json:"parameters,omitempty"`
}

// ParameterInformation represents a parameter of a callable-signature. A parameter can
// have a label and a doc-comment.
type ParameterInformation struct {
	Label         string `json:"label"`
	Documentation string `json:"documentation,omitempty"`
}

// CancelParams is the params send to ‘$/cancelRequest’ method
type CancelParams struct {
	ID jsonrpc2.ID `json:"id"`
}

// DidOpenTextDocumentParams is sent from client to the server when the document that was opened.
type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams is sent from client to the server to signal changes to a text document
type DidChangeTextDocumentParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// TextDocumentContentChangeEvent an event describing a change to a text document. If range and rangeLength are omitted
// the new text is considered to be the full content of the document.
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitEmpty"`
	RangeLength uint   `json:"rangeLength,omitEmpty"`
	Text        string `json:"text"`
}

// DidCloseTextDocumentParams is sent from the client to the server when the document got closed in the client.
type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidSaveTextDocumentParams is sent from the client to the server when the document was saved in the client.
type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

// WillSaveTextDocumentParams is sent from the client to the server before the document is actually saved.
type WillSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Reason       TextDocumentSaveReason `json:"reason,omitempty"`
}

// Hover is the result of a hover request.
type Hover struct {
	Contents []MarkedString `json:"contents"`
	Range    *Range         `json:"range,omitempty"`
}

// CompletionParams is the struct for parameters send to "textDocument/completion" request
type CompletionParams struct {
	TextDocumentPositionParams
	Context CompletionContext `json:"context,omitempty"`
}

// CompletionContext Contains additional information about the context in which a completion request is triggered.
type CompletionContext struct {
	TriggerKind      CompletionTriggerKind `json:"triggerKind"`
	TriggerCharacter string                `json:"triggerCharacter,omitempty"`
}

// CompletionList Represents a collection of [completion items](#CompletionItem) to be presented
// in the editor.
type CompletionList struct {
	IsIncomplete bool              `json:"isIncomplete"`
	Items        []*CompletionItem `json:"items"`
}

// CompletionItem represents The completion items.
type CompletionItem struct {
	Label            string             `json:"label"`
	Kind             CompletionItemKind `json:"kind,omitempty"`
	Detail           string             `json:"detail,omitempty"`
	Documentation    string             `json:"documentation,omitempty"`
	SortText         string             `json:"sortText,omitempty"`
	FilterText       string             `json:"filterText,omitempty"`
	InsertText       string             `json:"insertText,omitempty"`
	InsertTextFormat InsertTextFormat   `json:"insertTextFormat,omitempty"`
	TextEdit         *TextEdit          `json:"textEdit,omitempty"`
	Data             interface{}        `json:"data,omitempty"`
}

// PublishDiagnosticsParams is the params sent from the server to the client for textDocument/publishDiagnostics method
type PublishDiagnosticsParams struct {
	URI         DocumentURI   `json:"uri"`
	Diagnostics []*Diagnostic `json:"diagnostics"`
}

// DocumentFormattingParams is the params sent from the client for textDocument/formatting request
type DocumentFormattingParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Options      FormattingOptions      `json:"options"`
}

// FormattingOptions represent Value-object describing what options formatting should use.
type FormattingOptions struct {
	TabSize      int    `json:"tabSize"`
	InsertSpaces bool   `json:"insertSpaces"`
	Key          string `json:"key"`
}

// ReferenceParams is the params sent from the client for textDocument/references request
type ReferenceParams struct {
	*TextDocumentPositionParams

	Context ReferenceContext `json:"context"`
}

// ReferenceContext is the context used in ReferenceParams
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// RenameParams is the params sent from the client for textDocument/rename request
type RenameParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
	NewName      string                 `json:"newName"`
}

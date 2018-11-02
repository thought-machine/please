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
	SignatureHelpProvider      *SignatureHelpOptions `json:"signatureHelpOptions"`
	DefinitionProvider         bool                  `json:"definitionProvider"`
	TypeDefinitionProvider     bool                  `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider     bool                  `json:"implementationProvider,omitempty"`
	ReferenceProvider          bool                  `json:"referenceProvider,omitempty"`
	DocumentSymbolProvider     bool                  `json:"documentSymbolProvider,omitempty"`
	DocumentHighlightProvider  bool                  `json:"documentHighlightProvider,omitempty"`
	DocumentFormattingProvider bool                  `json:"documentFormattingProvider,omitempty"`
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

type DidCloseTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

type DidSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Text         string                 `json:"text,omitempty"`
}

type WillSaveTextDocumentParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Reason       TextDocumentSaveReason `json:"reason,omitempty"`
}

// Hover is the result of a hover request.
type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
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

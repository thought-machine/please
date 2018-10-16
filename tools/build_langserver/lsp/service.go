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

// Hover is the result of a hover request.
type Hover struct {
	Contents MarkupContent  `json:"contents"`
	Range    *Range         `json:"range,omitempty"`
}

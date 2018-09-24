package lsp

type ServerCapabilities struct {
	/**
	 * Defines how text documents are synced. Is either a detailed structure defining each notification or
	 * for backwards compatibility the TextDocumentSyncKind number. If omitted it defaults to `TextDocumentSyncKind.None`.
	 */
	TextDocumentSync   		  *TextDocumentSyncKind   `json:"textDocumentSync,omitempty"`


	HoverProvider 			   bool 						 `json:"hoverProvider"`
	CompletionProvider  	   *CompletionOptions         `json:"completionProvider"`
	SignatureHelpProvider 	   *SignatureHelpOptions `json:"signatureHelpOptions"`
	DefinitionProvider         bool 		`json:"definitionProvider"`
	TypeDefinitionProvider     bool        `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider     bool 		 `json:"implementationProvider,omitempty"`
	ReferenceProvider		   bool 		 `json:"referenceProvider,omitempty"`
	DocumentSymbolProvider 	   bool			 `json:"documentSymbolProvider,omitempty"`
	DocumentHighlightProvider  bool 		 `json:"documentHighlightProvider"`
	DocumentFormattingProvider bool

}

type TextDocumentSyncOptions struct {
	OpenClose 			bool 		 `json:"openClose"`
	Change 	  			TextDocumentSyncKind  		 `json:"change"`
	WillSave  			bool 		 `json:"willSave,omitempty"`
	WillSaveWaitUtil	bool 		 `json:"willSaveWaitUntil"`
	Save 				*SaveOptions `json:"save"`
}

type TextDocumentSyncKind int

const (
	SyncNone        TextDocumentSyncKind = 0
	SyncFull        TextDocumentSyncKind = 1
	SyncIncremental TextDocumentSyncKind = 2
)

type SaveOptions struct {
	IncludeText bool `json:"includeText"`
}

type CompletionOptions struct {
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type SignatureHelpOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}
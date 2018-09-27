package lsp

import "github.com/sourcegraph/jsonrpc2"

type ServerCapabilities struct {
	/**
	 * Defines how text documents are synced. Is either a detailed structure defining each notification or
	 * for backwards compatibility the TextDocumentSyncKind number. If omitted it defaults to `TextDocumentSyncKind.None`.
	 */
	TextDocumentSync   		  *TextDocumentSyncKind   `json:"textDocumentSync,omitempty"`


	HoverProvider 			   bool 						 `json:"hoverProvider"`
	CompletionProvider  	   *CompletionOptions        	 `json:"completionProvider"`
	SignatureHelpProvider 	   *SignatureHelpOptions 		 `json:"signatureHelpOptions"`
	DefinitionProvider         bool 						 `json:"definitionProvider"`
	TypeDefinitionProvider     bool        					 `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider     bool 		 				 `json:"implementationProvider,omitempty"`
	ReferenceProvider		   bool 		 				 `json:"referenceProvider,omitempty"`
	DocumentSymbolProvider 	   bool			 				 `json:"documentSymbolProvider,omitempty"`
	DocumentHighlightProvider  bool 		 				 `json:"documentHighlightProvider,omitempty"`
	DocumentFormattingProvider bool							 `json:"documentFormattingProvider,omitempty"`

}

// TODO(bnm): this might not be needed
type TextDocumentSyncOptions struct {
	OpenClose 			bool 		 				 `json:"openClose"`
	Change 	  			*TextDocumentSyncKind  		 `json:"change"`
	WillSave  			bool 		 				 `json:"willSave,omitempty"`
	WillSaveWaitUtil	bool 		 		   		 `json:"willSaveWaitUntil"`
	Save 				*SaveOptions 				 `json:"save"`
}


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

type CancelParams struct {
	ID jsonrpc2.ID `json:"id"`
}

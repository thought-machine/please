; Definition of Please mode for Emacs.
; Inherits from python-mode for formatting but uses lsp-mode for
; autocompletion etc.
;
; Add this to .emacs to make this load automatically.
; (add-to-list 'auto-mode-alist '("/BUILD\\'" . please-mode))
; (add-to-list 'auto-mode-alist '("/BUILD.plz\\'" . please-mode))

(require 'lsp)

(define-derived-mode please-mode python-mode "plz")

(lsp-register-client
 (make-lsp-client :new-connection (lsp-stdio-connection '("/home/peter/git/please/plz-out/bin/tools/build_langserver/build_langserver" "--log_file=/tmp/plzl.log" "-v=4"))
                  :major-modes '(please-mode)
                  :server-id 'plz))

(defun please-mode-fmt-on-save ()
  (interactive)
  (when (eq major-mode 'please-mode) (lsp-format-buffer)))

(add-hook 'before-save-hook 'please-mode-fmt-on-save)
(add-to-list 'lsp-language-id-configuration '(please-mode . "plz"))
(add-hook 'please-mode-hook #'lsp)

(add-to-list 'auto-mode-alist '("/BUILD\\'" . please-mode))
(add-to-list 'auto-mode-alist '("/BUILD.plz\\'" . please-mode))

(provide 'please-mode)

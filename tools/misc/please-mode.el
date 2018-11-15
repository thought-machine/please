; Definition of Please mode for Emacs.
; Inherits from python-mode for formatting but uses lsp-mode for
; autocompletion etc.
;
; Add this to .emacs to make this load automatically.
; (add-to-list 'auto-mode-alist '("/BUILD\\'" . please-mode))
; (add-to-list 'auto-mode-alist '("/BUILD.plz\\'" . please-mode))
; And add this to run buildifier on the current buffer when saving:
; (add-hook 'after-save-hook 'please-buildify-on-save).
; TODO(peterebden): get rid of please-buildify-on-save once plz tool langserver
;                   supports the textDocument/formatting method

(require 'lsp-mode)

(defun please-buildify ()
  "Format the current buffer according to the buildifier tool,
  in a pretty quick and dirty way."
  (interactive)
  (call-process "buildifier" nil nil nil "-mode=fix" (buffer-file-name))
  (revert-buffer t t))

;;;###autoload
(defun please-buildify-on-save ()
  (interactive)
  (when (eq major-mode 'please-mode) (please-buildify)))

(require 'lsp-imenu)
(add-hook 'lsp-after-open-hook 'lsp-enable-imenu)
(use-package lsp-ui
  :ensure t
  :config
  (setq lsp-ui-sideline-ignore-duplicate t)
  (add-hook 'lsp-mode-hook 'lsp-ui-mode))

(lsp-define-stdio-client
 please-mode
 "Please"
 (lambda () default-directory) ; TODO(peterebden): what should we do here?
 '("plz" "tool" "langserver"))

(define-derived-mode please-mode python-mode "Please mode")

(add-hook 'please-mode-hook
          (lambda ()
           (please-mode-enable)))

(provide 'please-mode)

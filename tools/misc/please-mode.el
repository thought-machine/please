; Definition of Please mode for Emacs.
; Inherits from python-mode for formatting but uses lsp-mode for
; autocompletion etc.
;
; Add this to .emacs to make this load automatically.
; (add-to-list 'auto-mode-alist '("/BUILD\\'" . please-mode))
; (add-to-list 'auto-mode-alist '("/BUILD.plz\\'" . please-mode))

(require 'lsp-mode)
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

(defun please-mode-fmt-on-save ()
  (interactive)
  (when (eq major-mode 'please-mode) (lsp-format-buffer)))

(add-hook 'before-save-hook 'please-mode-fmt-on-save)

(provide 'please-mode)

; Definition of Please mode for Emacs.
; Mostly inherits from python-mode and has that do all the hard work, the
; main advantage is automatic invocation of buildifier.
;
; Add this to .emacs to make this load automatically.
;(add-to-list 'auto-mode-alist '("BUILD" . please-mode))
; And add this to run buildifier on the current buffer when saving:
; (add-hook 'after-save-hook 'please-buildify-on-save).

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

(define-derived-mode please-mode python-mode "Please mode")

(provide 'please-mode)

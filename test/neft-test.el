;;; neft-test.el --- Tests for neft -*- lexical-binding: t; -*-

(require 'ert)
(require 'neft)

(ert-deftest neft-process-args-include-roots-and-thresholds ()
  (let ((neft-directories '("/tmp/a" "/tmp/b"))
        (neft-recursive t)
        (neft-many-results-threshold 12)
        (neft-snippets-when-many 1)
        (neft-snippets-when-few 4))
    (should (equal (neft--process-args "foo bar")
                   '("search" "--query=foo bar" "--format" "json"
                     "--recursive=true"
                     "--many-threshold" "12"
                     "--snippets-when-many" "1"
                     "--snippets-when-few" "4"
                     "--root" "/tmp/a"
                     "--root" "/tmp/b")))))

(ert-deftest neft-process-args-preserve-empty-query ()
  (let ((neft-directories '("/tmp/a")))
    (should (equal (neft--process-args "")
                   '("search" "--query=" "--format" "json"
                     "--recursive=true"
                     "--many-threshold" "50"
                     "--snippets-when-many" "1"
                     "--snippets-when-few" "5"
                     "--root" "/tmp/a")))))

(ert-deftest neft-render-results-highlights-match-ranges ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "kensaku")
    (neft--render-results
     '((query . "kensaku")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 1)
                  (snippets . (((line . 3)
                                (text . "検索 test")
                                (matches . (((start . 0) (end . 2))))))))))))
    (goto-char (point-min))
    (should (search-forward "alpha" nil t))
    (should (search-forward "検索" nil t))
    (let ((face (get-text-property (match-beginning 0) 'face)))
      (should (or (eq face 'neft-match-face)
                  (memq 'neft-match-face face))))))

(ert-deftest neft-handle-output-keeps-results-in-neft-buffer ()
  (let ((output (generate-new-buffer " *neft-test-output*")))
    (unwind-protect
        (with-temp-buffer
          (neft-mode)
          (setq neft--query "")
          (with-current-buffer output
            (insert "{\"query\":\"\",\"files\":[{\"path\":\"/tmp/a.org\",\"title\":\"alpha\",\"modified\":\"2026-01-01T00:00:00Z\",\"match_count\":0,\"snippets\":null}]}"))
          (neft--handle-output output)
          (should neft--results)
          (goto-char (point-min))
          (should (search-forward "alpha" nil t))
          (should-not (search-forward "No matches" nil t)))
      (when (buffer-live-p output)
        (kill-buffer output)))))

(ert-deftest neft-query-markers-track-end-insertion ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "")
    (neft--render-empty)
    (goto-char (marker-position neft--query-end))
    (insert "kensaku")
    (should (equal neft--query "kensaku"))
    (should (= (marker-position neft--query-end)
               (+ (marker-position neft--query-start) 7)))
    (when neft--timer
      (cancel-timer neft--timer)
      (setq neft--timer nil))))

(ert-deftest neft-query-accepts-self-insert-and-dwim-keys ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "")
    (neft--render-empty)
    (goto-char (marker-position neft--query-end))
    (should (eq (key-binding (kbd "k")) 'self-insert-command))
    (insert "ki")
    (call-interactively #'neft-refresh-or-insert)
    (call-interactively #'neft-quit-or-insert)
    (should (equal neft--query "kigq"))
    (call-interactively #'neft-delete-backward-char)
    (should (equal neft--query "kig"))
    (when neft--timer
      (cancel-timer neft--timer)
      (setq neft--timer nil))))

(ert-deftest neft-open-result-visits-file-line ()
  (let ((file (make-temp-file "neft" nil ".org" "one\ntwo\nthree\n")))
    (unwind-protect
        (with-temp-buffer
          (neft-mode)
          (let ((inhibit-read-only t))
            (insert (propertize "result" 'neft-path file 'neft-line 2)))
          (goto-char (point-min))
          (neft-open-result)
          (should (equal (buffer-file-name) file))
          (should (= (line-number-at-pos) 2))
          (kill-buffer))
      (ignore-errors (delete-file file)))))

(provide 'neft-test)

;;; neft-test.el ends here

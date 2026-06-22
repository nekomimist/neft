;;; neft-test.el --- Tests for neft -*- lexical-binding: t; -*-

(require 'ert)
(require 'neft)

(ert-deftest neft-process-args-include-roots-and-thresholds ()
  (let ((neft-directories '("/tmp/a" "/tmp/b"))
        (neft-recursive t)
        (neft-file-extensions '("org"))
        (neft-many-results-threshold 12)
        (neft-snippets-when-many 1)
        (neft-snippets-when-few 4))
    (should (equal (neft--process-args "foo bar")
                   '("search" "--query=foo bar" "--format" "json"
                     "--recursive=true"
                     "--many-threshold" "12"
                     "--snippets-when-many" "1"
                     "--snippets-when-few" "4"
                     "--extension" "org"
                     "--root" "/tmp/a"
                     "--root" "/tmp/b")))))

(ert-deftest neft-process-args-preserve-empty-query ()
  (let ((neft-directories '("/tmp/a"))
        (neft-file-extensions '("org")))
    (should (equal (neft--process-args "")
                   '("search" "--query=" "--format" "json"
                     "--recursive=true"
                     "--many-threshold" "50"
                     "--snippets-when-many" "1"
                     "--snippets-when-few" "5"
                     "--extension" "org"
                     "--root" "/tmp/a")))))

(ert-deftest neft-process-args-include-file-extensions ()
  (let ((neft-directories '("/tmp/a"))
        (neft-file-extensions '("org" ".txt" "")))
    (should (equal (neft--process-args "memo")
                   '("search" "--query=memo" "--format" "json"
                     "--recursive=true"
                     "--many-threshold" "50"
                     "--snippets-when-many" "1"
                     "--snippets-when-few" "5"
                     "--extension" "org"
                     "--extension" ".txt"
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

(ert-deftest neft-render-results-highlights-indented-match-ranges ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "needle")
    (neft--render-results
     '((query . "needle")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 1)
                  (snippets . (((line . 3)
                                (text . "  needle")
                                (matches . (((start . 2) (end . 8))))))))))))
    (goto-char (point-min))
    (should (search-forward "  needle" nil t))
    (let ((face (get-text-property (+ (match-beginning 0) 2) 'face)))
      (should (or (eq face 'neft-match-face)
                  (memq 'neft-match-face face))))))

(ert-deftest neft-render-results-keeps-files-spaced-not-snippets ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "needle")
    (neft--render-results
     '((query . "needle")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 2)
                  (snippets . (((line . 1)
                                (text . "first needle")
                                (matches . (((start . 6) (end . 12)))))
                               ((line . 2)
                                (text . "second needle")
                                (matches . (((start . 7) (end . 13))))))))
                 ((path . "/tmp/b.org")
                  (title . "beta")
                  (match_count . 1)
                  (snippets . (((line . 3)
                                (text . "third needle")
                                (matches . (((start . 6) (end . 12))))))))))))
    (should (string-match-p
             "first needle\n    2: second needle\n\nbeta"
             (buffer-string)))
    (should-not (string-match-p
                 "first needle\n\n    2: second needle"
                 (buffer-string)))))

(ert-deftest neft-render-results-keeps-empty-query-files-single-spaced ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "")
    (neft--render-results
     '((query . "")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 0)
                  (snippets . nil))
                 ((path . "/tmp/b.org")
                  (title . "beta")
                  (match_count . 0)
                  (snippets . nil))))))
    (should (string-match-p
             "alpha\n/tmp/a.org\n\nbeta"
             (buffer-string)))
    (should-not (string-match-p
                 "alpha\n/tmp/a.org\n\n\nbeta"
                 (buffer-string)))))

(ert-deftest neft-handle-output-keeps-results-in-neft-buffer ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "")
    (neft--handle-output
     "{\"query\":\"\",\"files\":[{\"path\":\"/tmp/a.org\",\"title\":\"alpha\",\"modified\":\"2026-01-01T00:00:00Z\",\"match_count\":0,\"snippets\":null}]}")
    (should neft--results)
    (goto-char (point-min))
    (should (search-forward "alpha" nil t))
    (should-not (search-forward "No matches" nil t))))

(ert-deftest neft-process-filter-keeps-output-out-of-neft-buffer ()
  (let ((process (start-process "neft-test-sleep" nil "sleep" "5")))
    (unwind-protect
        (with-temp-buffer
          (neft-mode)
          (setq neft--query "")
          (neft--render-empty)
          (process-put process 'neft-output "")
          (neft--process-filter process "{\"query\":\"")
          (neft--process-filter process "\"}")
          (should (equal (process-get process 'neft-output)
                         "{\"query\":\"\"}"))
          (should (equal (buffer-string) "Search: \n\n")))
      (when (process-live-p process)
        (delete-process process)))))

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

(ert-deftest neft-query-is-the-only-writable-area ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "abc")
    (neft--render-results
     '((query . "abc")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 1)
                  (snippets . (((line . 1)
                                (text . "abc")
                                (matches . (((start . 0) (end . 3))))))))))))
    (let ((original (buffer-string)))
      (goto-char (point-min))
      (should-error (insert "x") :type 'text-read-only)
      (should-error (insert "x") :type 'text-read-only)
      (should (equal (buffer-string) original)))
    (goto-char (marker-position neft--query-end))
    (insert "d")
    (should (equal neft--query "abcd"))
    (goto-char (point-min))
    (search-forward "alpha")
    (let ((original (buffer-string)))
      (should-error (insert "x") :type 'text-read-only)
      (should-error (insert "x") :type 'text-read-only)
      (should (equal (buffer-string) original)))
    (goto-char (point-max))
    (let ((original (buffer-string)))
      (should-error (insert "x") :type 'text-read-only)
      (should-error (insert "x") :type 'text-read-only)
      (should (equal (buffer-string) original)))
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

(ert-deftest neft-move-beginning-of-line-stops-at-query-start ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "abcdef")
    (neft--render-results
     '((query . "abcdef")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 1)
                  (snippets . (((line . 12)
                                (text . "abcdef")
                                (matches . (((start . 0) (end . 6))))))))))))
    (goto-char (marker-position neft--query-end))
    (neft-move-beginning-of-line)
    (should (= (point) (marker-position neft--query-start)))
    (goto-char (point-min))
    (search-forward "abcdef" nil nil 2)
    (neft-move-beginning-of-line)
    (should (= (current-column) 0))))

(ert-deftest neft-kill-line-preserves-query-row-newline-at-empty-query ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "")
    (neft--render-results
     '((query . "")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 0)
                  (snippets . nil))))))
    (let ((original (buffer-string)))
      (goto-char (marker-position neft--query-end))
      (neft-kill-line)
      (should (equal (buffer-string) original))
      (should (equal neft--query "")))))

(ert-deftest neft-kill-line-kills-query-text-only ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "abcdef")
    (neft--render-results
     '((query . "abcdef")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 1)
                  (snippets . nil))))))
    (goto-char (+ (marker-position neft--query-start) 2))
    (neft-kill-line)
    (should (equal neft--query "ab"))
    (should (string-match-p "\\`Search: ab\n\nalpha" (buffer-string)))
    (when neft--timer
      (cancel-timer neft--timer)
      (setq neft--timer nil))))

(ert-deftest neft-file-navigation-moves-forward-by-file ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "needle")
    (neft--render-results
     '((query . "needle")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 1)
                  (snippets . (((line . 1)
                                (text . "needle")
                                (matches . (((start . 0) (end . 6))))))))
                 ((path . "/tmp/b.org")
                  (title . "beta")
                  (match_count . 1)
                  (snippets . nil))))))
    (goto-char (marker-position neft--query-end))
    (neft-next-file)
    (should (looking-at-p "alpha"))
    (neft-next-file)
    (should (looking-at-p "beta"))
    (let ((last-file (point)))
      (neft-next-file)
      (should (= (point) last-file)))))

(ert-deftest neft-file-navigation-moves-backward-by-file ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "needle")
    (neft--render-results
     '((query . "needle")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 1)
                  (snippets . nil))
                 ((path . "/tmp/b.org")
                  (title . "beta")
                  (match_count . 1)
                  (snippets . (((line . 1)
                                (text . "needle")
                                (matches . (((start . 0) (end . 6))))))))))))
    (goto-char (point-min))
    (search-forward "needle" nil nil 2)
    (neft-previous-file)
    (should (looking-at-p "beta"))
    (neft-previous-file)
    (should (looking-at-p "alpha"))
    (let ((first-file (point)))
      (neft-previous-file)
      (should (= (point) first-file)))))

(ert-deftest neft-mode-disables-completion-preview ()
  (with-temp-buffer
    (when (fboundp 'completion-preview-mode)
      (completion-preview-mode 1))
    (neft-mode)
    (should (null completion-at-point-functions))
    (when (boundp 'completion-preview-mode)
      (should-not completion-preview-mode))))

(ert-deftest neft-restores-window-configuration-on-quit ()
  (let ((left (get-buffer-create " *neft-test-left*"))
        (right (get-buffer-create " *neft-test-right*"))
        (neft-buffer (get-buffer-create "*neft*"))
        (neft-restore-window-configuration t))
    (unwind-protect
        (progn
          (delete-other-windows)
          (switch-to-buffer left)
          (split-window-right)
          (other-window 1)
          (switch-to-buffer right)
          (other-window -1)
          (cl-letf (((symbol-function 'neft-refresh) #'ignore))
            (neft))
          (should (= (length (window-list)) 1))
          (should (eq (current-buffer) neft-buffer))
          (neft-quit)
          (should (= (length (window-list)) 2))
          (should (get-buffer-window left))
          (should (get-buffer-window right)))
      (when (get-buffer "*neft*")
        (kill-buffer "*neft*"))
      (kill-buffer left)
      (kill-buffer right)
      (delete-other-windows))))

(ert-deftest neft-restores-window-configuration-on-buffer-kill ()
  (let ((left (get-buffer-create " *neft-test-left*"))
        (right (get-buffer-create " *neft-test-right*"))
        (neft-restore-window-configuration t))
    (unwind-protect
        (progn
          (delete-other-windows)
          (switch-to-buffer left)
          (split-window-right)
          (other-window 1)
          (switch-to-buffer right)
          (other-window -1)
          (cl-letf (((symbol-function 'neft-refresh) #'ignore))
            (neft))
          (should (= (length (window-list)) 1))
          (kill-buffer "*neft*")
          (should (= (length (window-list)) 2))
          (should (get-buffer-window left))
          (should (get-buffer-window right)))
      (when (get-buffer "*neft*")
        (kill-buffer "*neft*"))
      (kill-buffer left)
      (kill-buffer right)
      (delete-other-windows))))

(ert-deftest neft-render-results-preserves-query-point-offset ()
  (with-temp-buffer
    (neft-mode)
    (setq neft--query "abcdef")
    (neft--render-empty)
    (goto-char (+ (marker-position neft--query-start) 2))
    (neft--render-results
     '((query . "abcdef")
       (files . (((path . "/tmp/a.org")
                  (title . "alpha")
                  (match_count . 1)
                  (snippets . nil))))))
    (should (= (point)
               (+ (marker-position neft--query-start) 2)))))

(ert-deftest neft-start-search-invalidates-killed-process-before-sentinel ()
  (with-temp-buffer
    (neft-mode)
    (let ((neft-directories '("/tmp"))
          (started-generations nil))
      (let ((original-make-process (symbol-function 'make-process)))
        (cl-letf (((symbol-function 'make-process)
                   (lambda (&rest _args)
                     (let ((process (funcall original-make-process
                                             :name "neft-test-sleep"
                                             :buffer nil
                                             :command '("sleep" "5"))))
                       (push neft--generation started-generations)
                       process))))
          (neft--start-search)
          (let ((first-process neft--process)
                (first-generation neft--generation))
            (neft--start-search)
            (should (> neft--generation first-generation))
            (should-not (process-live-p first-process))
            (should (equal (nreverse started-generations) '(1 2)))
            (when (process-live-p neft--process)
              (delete-process neft--process))))))))

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

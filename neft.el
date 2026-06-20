;;; neft.el --- Fast Denote-oriented org search UI -*- lexical-binding: t; -*-

;; Copyright (C) 2026 nekomimist
;; SPDX-License-Identifier: MIT

;; Author: nekomimist
;; Version: 0.1.0
;; Package-Requires: ((emacs "28.1"))
;; Keywords: convenience, outlines, files
;; URL: https://github.com/nekomimist/neft

;;; Commentary:

;; neft provides a Deft-inspired search buffer backed by an external Go
;; executable.  It is designed for org files, including Denote note trees.

;;; Code:

(require 'cl-lib)
(require 'json)
(require 'subr-x)

(defgroup neft nil
  "Fast org search UI backed by the neft executable."
  :group 'convenience)

(defcustom neft-directories nil
  "Directories or org files searched by `neft'."
  :type '(repeat directory)
  :group 'neft)

(defcustom neft-recursive t
  "Whether `neft' searches child directories."
  :type 'boolean
  :group 'neft)

(defcustom neft-program "neft"
  "Path to the neft executable."
  :type 'file
  :group 'neft)

(defcustom neft-debounce-seconds 0.15
  "Seconds to wait after input before starting a search."
  :type 'number
  :group 'neft)

(defcustom neft-many-results-threshold 50
  "File-count threshold where neft switches to compact snippets."
  :type 'integer
  :group 'neft)

(defcustom neft-snippets-when-many 1
  "Snippet count per file when many files match."
  :type 'integer
  :group 'neft)

(defcustom neft-snippets-when-few 5
  "Snippet count per file when fewer files match."
  :type 'integer
  :group 'neft)

(defface neft-query-face
  '((t :inherit minibuffer-prompt))
  "Face for the query label."
  :group 'neft)

(defface neft-file-face
  '((t :inherit font-lock-function-name-face :weight bold))
  "Face for file result headers."
  :group 'neft)

(defface neft-path-face
  '((t :inherit shadow))
  "Face for file paths."
  :group 'neft)

(defface neft-match-face
  '((t :inherit isearch))
  "Face for highlighted matches."
  :group 'neft)

(defvar-local neft--query "")
(defvar-local neft--process nil)
(defvar-local neft--timer nil)
(defvar-local neft--generation 0)
(defvar-local neft--results nil)
(defvar-local neft--query-start nil)
(defvar-local neft--query-end nil)

(defvar neft-mode-map
  (let ((map (make-sparse-keymap)))
    (define-key map (kbd "RET") #'neft-return-dwim)
    (define-key map (kbd "DEL") #'neft-delete-backward-char)
    (define-key map (kbd "C-h") #'neft-delete-backward-char)
    (define-key map (kbd "g") #'neft-refresh-or-insert)
    (define-key map (kbd "q") #'neft-quit-or-insert)
    (define-key map (kbd "C-c C-k") #'neft-clear-query)
    map)
  "Keymap for `neft-mode'.")

;;;###autoload
(define-derived-mode neft-mode fundamental-mode "neft"
  "Major mode for the neft search buffer."
  (setq-local buffer-read-only nil)
  (setq-local truncate-lines t)
  (add-hook 'after-change-functions #'neft--after-change nil t))

;;;###autoload
(defun neft ()
  "Open the neft search buffer."
  (interactive)
  (let ((buffer (get-buffer-create "*neft*")))
    (pop-to-buffer buffer)
    (unless (derived-mode-p 'neft-mode)
      (neft-mode)
      (neft--render-empty))
    (goto-char neft--query-end)
    (neft-refresh)))

(defun neft-refresh ()
  "Run a neft search for the current query."
  (interactive)
  (neft--start-search))

(defun neft-return-dwim ()
  "Refresh from the query row or open the result at point."
  (interactive)
  (if (neft--in-query-p)
      (neft-refresh)
    (neft-open-result)))

(defun neft-delete-backward-char ()
  "Delete backward in the query row."
  (interactive)
  (if (neft--in-query-p)
      (call-interactively #'delete-backward-char)
    (user-error "Move to the search row to edit the query")))

(defun neft-refresh-or-insert ()
  "Insert g in the query row, otherwise refresh results."
  (interactive)
  (if (neft--in-query-p)
      (insert "g")
    (neft-refresh)))

(defun neft-quit-or-insert ()
  "Insert q in the query row, otherwise quit the neft window."
  (interactive)
  (if (neft--in-query-p)
      (insert "q")
    (quit-window)))

(defun neft-clear-query ()
  "Clear the neft query."
  (interactive)
  (let ((inhibit-read-only t))
    (delete-region (marker-position neft--query-start)
                   (marker-position neft--query-end))
    (setq neft--query ""
          neft--query-end (copy-marker neft--query-start t))
    (neft--schedule-search)))

(defun neft-open-result ()
  "Open the result at point."
  (interactive)
  (let ((path (get-text-property (point) 'neft-path))
        (line (get-text-property (point) 'neft-line)))
    (unless path
      (user-error "No neft result at point"))
    (find-file path)
    (when line
      (goto-char (point-min))
      (forward-line (1- line)))))

(defun neft--in-query-p (&optional position)
  (let ((position (or position (point))))
    (and (markerp neft--query-start)
         (<= (marker-position neft--query-start) position)
         (= (line-number-at-pos position) 1))))

(defun neft--after-change (beg _end _len)
  (when (and (markerp neft--query-start)
             (>= beg (marker-position neft--query-start))
             (= (line-number-at-pos beg) 1))
    (save-excursion
      (goto-char (marker-position neft--query-start))
      (set-marker neft--query-end (line-end-position)))
    (setq neft--query
          (buffer-substring-no-properties
           (marker-position neft--query-start)
           (marker-position neft--query-end)))
    (neft--schedule-search)))

(defun neft--schedule-search ()
  (when neft--timer
    (cancel-timer neft--timer))
  (setq neft--timer
        (run-at-time neft-debounce-seconds nil
                     (lambda (buffer)
                       (when (buffer-live-p buffer)
                         (with-current-buffer buffer
                           (neft--start-search))))
                     (current-buffer))))

(defun neft--start-search ()
  (unless neft-directories
    (user-error "Set `neft-directories' before using neft"))
  (cl-incf neft--generation)
  (when (process-live-p neft--process)
    (delete-process neft--process))
  (let* ((generation neft--generation)
         (output-buffer (generate-new-buffer " *neft-output*"))
         (args (neft--process-args neft--query)))
    (setq neft--process
          (make-process
           :name "neft"
           :buffer output-buffer
           :command (cons neft-program args)
           :noquery t
           :sentinel #'neft--process-sentinel))
    (process-put neft--process 'neft-buffer (current-buffer))
    (process-put neft--process 'neft-generation generation)))

(defun neft--process-sentinel (process event)
  (when (memq (process-status process) '(exit signal))
    (let ((buffer (process-get process 'neft-buffer))
          (out (process-buffer process))
          (generation (process-get process 'neft-generation))
          (status (process-exit-status process)))
      (when (buffer-live-p buffer)
        (with-current-buffer buffer
          (when (= generation neft--generation)
            (if (= status 0)
                (neft--handle-output out)
              (neft--render-error event out))))))
    (when (buffer-live-p (process-buffer process))
      (kill-buffer (process-buffer process)))))

(defun neft--process-args (query)
  (append
   (list "search"
         (format "--query=%s" query)
         "--format" "json"
         (format "--recursive=%s" (if neft-recursive "true" "false"))
         "--many-threshold" (number-to-string neft-many-results-threshold)
         "--snippets-when-many" (number-to-string neft-snippets-when-many)
         "--snippets-when-few" (number-to-string neft-snippets-when-few))
   (cl-mapcan (lambda (dir) (list "--root" (expand-file-name dir)))
              neft-directories)))

(defun neft--handle-output (output-buffer)
  (let ((result
         (let ((json-object-type 'alist)
               (json-array-type 'list)
               (json-key-type 'symbol))
           (with-current-buffer output-buffer
             (json-read-from-string (buffer-string))))))
    (setq neft--results result)
    (neft--render-results result)))

(defun neft--render-empty ()
  (let ((inhibit-read-only t)
        (inhibit-modification-hooks t))
    (erase-buffer)
    (insert (propertize "Search: " 'face 'neft-query-face))
    (setq neft--query-start (copy-marker (point)))
    (insert neft--query)
    (setq neft--query-end (copy-marker (point)))
    (insert "\n\n")))

(defun neft--render-results (result)
  (let ((query neft--query)
        (files (alist-get 'files result))
        (query-offset (and (neft--in-query-p)
                           (- (point) (marker-position neft--query-start)))))
    (let ((inhibit-read-only t)
          (old-path (get-text-property (point) 'neft-path))
          (old-line (get-text-property (point) 'neft-line)))
      (neft--render-empty)
      (if files
          (dolist (file files)
            (neft--insert-file file))
        (insert "No matches\n"))
      (setq neft--query query)
      (add-text-properties (1+ (marker-position neft--query-end)) (point-max)
                           '(read-only t rear-nonsticky t))
      (cond
       (query-offset
        (goto-char (min (+ (marker-position neft--query-start) query-offset)
                        (marker-position neft--query-end))))
       ((and old-path old-line)
        (neft--goto-result old-path old-line))))))

(defun neft--goto-result (path line)
  (goto-char (point-min))
  (let ((found nil))
    (while (and (not found)
                (text-property-search-forward 'neft-path path t))
      (when (equal (get-text-property (point) 'neft-line) line)
        (setq found t)))
    (unless found
      (goto-char (point-min)))))

(defun neft--insert-file (file)
  (let ((path (alist-get 'path file))
        (title (alist-get 'title file))
        (match-count (alist-get 'match_count file))
        (snippets (alist-get 'snippets file)))
    (let ((start (point)))
      (insert (propertize title 'face 'neft-file-face))
      (when (and match-count (> match-count 0))
        (insert (format " (%s)" match-count)))
      (insert "\n")
      (add-text-properties start (point) `(neft-path ,path neft-line 1)))
    (insert (propertize path 'face 'neft-path-face) "\n")
    (if snippets
        (dolist (snippet snippets)
          (neft--insert-snippet path snippet))
      nil)
    (insert "\n")))

(defun neft--insert-snippet (path snippet)
  (let* ((line (alist-get 'line snippet))
         (text (or (alist-get 'text snippet) ""))
         (matches (alist-get 'matches snippet))
         (start (point)))
    (insert (format "%5s: %s\n" line text))
    (add-text-properties start (point) `(neft-path ,path neft-line ,line))
    (dolist (range matches)
      (let* ((match-start (alist-get 'start range))
             (match-end (alist-get 'end range))
             (prefix-width (+ start 7))
             (beg (+ prefix-width match-start))
             (end (+ prefix-width match-end)))
        (when (and (<= start beg) (<= end (point)))
          (add-face-text-property beg end 'neft-match-face))))))

(defun neft--render-error (event output-buffer)
  (let ((message (if (buffer-live-p output-buffer)
                     (with-current-buffer output-buffer
                       (string-trim (buffer-string)))
                   "")))
    (let ((inhibit-read-only t))
      (neft--render-empty)
      (insert (format "Search failed: %s\n%s\n" (string-trim event) message)))))

(provide 'neft)

;;; neft.el ends here

.PHONY: test test-go test-el build clean

EMACS ?= emacs
GO ?= go

test: test-go test-el

test-go:
	$(GO) test ./...

test-el:
	$(EMACS) -Q --batch -L . -L test -l test/neft-test.el -f ert-run-tests-batch-and-exit

build:
	$(GO) build -buildvcs=false -o bin/neft ./cmd/neft

clean:
	rm -rf bin

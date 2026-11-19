# PKG selects the packages of `make test-pkg`.
PKG ?= ./...

test:
	go test ./... -count=1

test-pkg:
	go test $(PKG) -count=1

.PHONY: test test-pkg

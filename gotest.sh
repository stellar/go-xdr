#!/bin/bash
# The script does automatic checking on a Go package and its sub-packages, including:
# 1. goimports     (https://golang.org/x/tools/cmd/goimports)
# 2. go vet        (https://golang.org/cmd/vet)
# 3. test coverage (https://blog.golang.org/cover)

set -ex

test -z "$(goimports -l -w .)"
go vet ./...
go test -covermode=atomic -race ./...

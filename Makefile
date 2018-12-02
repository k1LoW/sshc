GO ?= GO111MODULE=on go

default: test

test:
	$(GO) test ./... -coverprofile=coverage.txt -covermode=count

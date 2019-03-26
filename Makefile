GO ?= GO111MODULE=on go

default: test

ci: test e2e

test:
	$(GO) test ./... -coverprofile=coverage.txt -covermode=count

e2e:
	./test/run.sh

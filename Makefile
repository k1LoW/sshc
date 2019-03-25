GO ?= GO111MODULE=on go

default: test

ci: test

test:
	$(GO) test ./... -coverprofile=coverage.txt -covermode=count

e2e:
	./test/run.sh

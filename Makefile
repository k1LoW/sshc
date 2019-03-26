export GO111MODULE=on

default: test

ci: test integration

test:
	go test ./... -coverprofile=coverage.txt -covermode=count

integration:
	chmod 600 ./testdata/id_rsa
	go test ./... -integration

.PHONY: default test

export GO111MODULE=on

default: test

ci: integration

test:
	go test ./... -coverprofile=coverage.txt -covermode=count

integration:
	chmod 600 ./testdata/id_rsa
	go test ./... -integration -coverprofile=coverage.txt -covermode=count

.PHONY: default test

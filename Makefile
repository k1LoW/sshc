export GO111MODULE=on

default: test

ci: depsdev test integration sec

test:
	go test -v ./... -coverprofile=coverage.txt -covermode=count

sec:
	gosec ./...

integration:
	chmod 600 ./testdata/id_rsa
	go test -v ./... -integration -coverprofile=coverage.txt -covermode=count

depsdev:
	go get github.com/linyows/git-semv/cmd/git-semv
	go get github.com/Songmu/ghch/cmd/ghch
	go get github.com/Songmu/gocredits/cmd/gocredits
	go get github.com/securego/gosec/cmd/gosec

.PHONY: default test

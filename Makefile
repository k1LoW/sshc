export GO111MODULE=on

default: test

ci: depsdev test integration sec

test:
	go test -v ./... -coverprofile=coverage.out -covermode=count

sec:
	gosec ./...

lint:
	golangci-lint run ./...

integration:
	chmod 600 ./testdata/id_rsa
	go test -v ./... -integration -coverprofile=coverage.out -covermode=count

depsdev:
	go get github.com/linyows/git-semv/cmd/git-semv
	go get github.com/Songmu/ghch/cmd/ghch
	go get github.com/Songmu/gocredits/cmd/gocredits
	go get github.com/securego/gosec/cmd/gosec

prerelease:
	git pull origin master --tag
	go mod tidy
	ghch -w -N ${VER}
	gocredits . > CREDITS
	git add CHANGELOG.md CREDITS go.mod
	git commit -m'Bump up version number'
	git tag ${VER}

.PHONY: default test

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
	go install github.com/linyows/git-semv/cmd/git-semv@v1.2.0
	go install github.com/Songmu/ghch/cmd/ghch@v0.10.2
	go install github.com/Songmu/gocredits/cmd/gocredits@v0.2.0
	go install github.com/securego/gosec/v2/cmd/gosec@v2.8.1

prerelease:
	git pull origin main --tag
	go mod tidy
	ghch -w -N ${VER}
	gocredits . > CREDITS
	git add CHANGELOG.md CREDITS go.mod
	git commit -m'Bump up version number'
	git tag ${VER}

.PHONY: default test

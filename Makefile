export GO111MODULE=on

default: test

ci: depsdev test integration

test:
	go test ./... -coverprofile=coverage.out -covermode=count

lint:
	golangci-lint run ./...

integration:
	chmod 600 ./testdata/id_rsa
	go test ./... -integration -coverprofile=coverage.out -covermode=count

depsdev:
	go install github.com/linyows/git-semv/cmd/git-semv@latest
	go install github.com/Songmu/ghch/cmd/ghch@latest
	go install github.com/Songmu/gocredits/cmd/gocredits@latest

prerelease:
	git pull origin main --tag
	go mod tidy
	ghch -w -N ${VER}
	gocredits . > CREDITS
	git add CHANGELOG.md CREDITS go.mod
	git commit -m'Bump up version number'
	git tag ${VER}

.PHONY: default test

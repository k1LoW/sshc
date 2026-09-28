export GO111MODULE=on

default: test

ci: depsdev check_license test integration

test:
	go test ./... -coverprofile=coverage.out -covermode=count

lint:
	golangci-lint run ./...

check_license:
	go-licenses check ./... --disallowed_types=permissive,forbidden,restricted --include_tests

integration:
	chmod 600 ./testdata/id_rsa
	go test -v ./... -integration -coverprofile=coverage.out -covermode=count

depsdev:
	go install github.com/linyows/git-semv/cmd/git-semv@latest
	go install github.com/Songmu/ghch/cmd/ghch@latest
	go install github.com/google/go-licenses/v2@latest

credits:
	go install github.com/Songmu/gocredits/cmd/gocredits@v1.0.0
	gocredits . > CREDITS

prerelease:
	git pull origin main --tag
	go mod tidy
	ghch -w -N ${VER}
	$(MAKE) credits
	git add CHANGELOG.md CREDITS go.mod
	git commit -m'Bump up version number'
	git tag ${VER}

prerelease_for_tagpr:
	$(MAKE) credits
	git add CHANGELOG.md CREDITS go.mod go.sum

.PHONY: default test credits

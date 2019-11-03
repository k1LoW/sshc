export GO111MODULE=on

default: test

ci: depsdev integration sec

test:
	go test -v ./... -coverprofile=coverage.txt -covermode=count

lint:
	golint ./...

sec:
	gosec ./...

integration:
	chmod 600 ./testdata/id_rsa
	go test -v ./... -integration -coverprofile=coverage.txt -covermode=count

depsdev:
	go get golang.org/x/lint/golint
	go get github.com/linyows/git-semv/cmd/git-semv
	go get github.com/Songmu/ghch/cmd/ghch
	go get github.com/Songmu/gocredits/cmd/gocredits
	go get github.com/securego/gosec/cmd/gosec
	go get github.com/aktau/github-release

prerelease:
	ghch -w -N ${VER}
	gocredits . > CREDITS
	git add CHANGELOG.md CREDITS
	git commit -m'Bump up version number'
	git tag ${VER}

release:
	github-release release --user k1LoW --repo sshc --tag ${shell git semv latest} --name ${shell git semv latest}

.PHONY: default test

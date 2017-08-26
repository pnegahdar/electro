PROJECT := electro
PKG := github.com/pnegahdar/$(PROJECT)
VERSION := $(shell git describe --tags --always --dirty)

all: build

yarnbuild:
	(cd fe/ && yarn install && yarn build)

bindata:
	go-bindata -o cmd/electro/staticfile.go  fe/build/...

build:
	mkdir -p bin/
	go build -i -o ./bin/local_build $(PKG)/cmd/$(PROJECT)
	cp ./bin/local_build $(GOPATH)/bin/$(PROJECT)-dev
	du -h ./bin/local_build

build-osx:
	GOOS=darwin GOARCH=amd64 go build -o ./bin/darwin_amd64 $(PKG)/cmd/$(PROJECT)

build-nix:
	GOOS=linux GOARCH=amd64 go build -o ./bin/linux_amd64 $(PKG)/cmd/$(PROJECT)


distribute: yarnbuild bindata build-nix build-osx

fmt:
	go fmt ./cmd/... ./pkg/...

test:
	go test -v -race ./cmd/... ./pkg/...

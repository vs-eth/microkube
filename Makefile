all: deps cmd generate build test

test:
	go test -race -run 'Test[^9]' -v ./...
	go test -race -run 'Test9' -v ./...

generate:
	go generate -v ./...

build:
	go build -ldflags="-s -w" github.com/uubk/microkube/cmd/microkubed

deps:
	dep ensure

cmd:
	./.check-deps.sh
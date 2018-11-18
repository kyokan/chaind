#!/bin/sh

cd $GOPATH/src/github.com/kyokan/chaind
make deps
GOOS=linux GOARCH=amd64 go build -ldflags '-w -extldflags "-static"' -o ./target/chaind ./cmd/chaind/main.go
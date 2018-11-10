deps:
	@echo "--> Running dep ensure..."
	@dep ensure -v

clean:
	packr clean
	rm -rf ./target

build:
	CGO_ENABLED=1 go build -o ./target/chaind ./cmd/chaind/main.go

install-global: build
	sudo mv ./target/chaind /usr/bin

test:
	go test -v ./...

.PHONY: build test
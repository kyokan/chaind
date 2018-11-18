deps:
	@echo "--> Running dep ensure..."
	@dep ensure -v

clean:
	rm -rf ./target

build:
	CGO_ENABLED=1 go build -o ./target/chaind ./cmd/chaind/main.go

build-cross:
	mkdir -p build/workspace
	cp -r cmd build/workspace
	cp -r internal build/workspace
	cp -r pkg build/workspace
	cp Makefile build/workspace
	cp Gopkg.lock build/workspace
	cp Gopkg.toml build/workspace
	docker build ./build -t chaind-cross-compilation:latest
	docker run --name chaind-cp-tmp chaind-cross-compilation:latest
	docker cp chaind-cp-tmp:/go/src/github.com/kyokan/chaind/target/chaind ./target/chaind-linux-amd64
	docker rm chaind-cp-tmp
	rm -rf build/workspace

install-global: build
	sudo mv ./target/chaind /usr/bin

test:
	go test -v ./...

package:
	@test -n "$(VERSION)" || (echo "version not specified"; exit 1)
	fpm -f -p target -s dir -t deb -n chaind -a amd64 -m "Kyokan, LLC <mslipper@kyokan.io>" \
    	--url "https://chaind.kyokan.io" \
    	--description "A security and caching layer for blockchain nodes." \
    	--license "AGPL-V3" \
        --vendor "Kyokan, LLC" \
		--config-files /etc/chaind/chaind.toml -v $(VERSION) \
		target/chaind-linux-amd64=/usr/bin/chaind \
		example/chaind.toml=/etc/chaind/

deploy:
	@test -n "$(VERSION)" || (echo "version not specified"; exit 1)
	@test -n "$(USERNAME)" || (echo "username not specified"; exit 1)
	@test -n "$(API_KEY)" || (echo "API key not specified"; exit 1)
	@test -n "$(GPG_PASSPHRASE)" || (echo "GPG passphrase not specified"; exit 1)
	@echo "--> Uploading version $(VERSION) to Bintray..."
	@curl -s -S -T ./target/chaind_$(VERSION)_amd64.deb -u$(USERNAME):$(API_KEY) \
		-H "X-GPG-PASSPHRASE: $(GPG_PASSPHRASE)" \
		-H "X-Bintray-Debian-Distribution: any" \
        -H "X-Bintray-Debian-Component: main" \
        -H "X-Bintray-Debian-Architecture: amd64" \
		https://api.bintray.com/content/kyokan/oss-deb/chaind/$(VERSION)/chaind_$(VERSION)_amd64.deb
	@sleep 3
	@echo "--> Publishing package..."
	@curl -s -S -X POST -u$(USERNAME):$(API_KEY) \
			https://api.bintray.com/content/kyokan/oss-deb/chaind/$(VERSION)/publish
	@sleep 10
	@echo "--> Forcing metadata calculation..."
	@curl -s -S -X POST -H "X-GPG-PASSPHRASE: $(GPG_PASSPHRASE)" -u$(USERNAME):$(API_KEY) https://api.bintray.com/calc_metadata/kyokan/oss-deb/

.PHONY: build build-cross test
.PHONY: build build-docker install clean test test-run test-write help default

BIN_NAME=cargo

VERSION := $(shell grep "const Version " version/version.go | sed -E 's/.*"(.+)"$$/\1/')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "+CHANGES" || true)
BUILD_DATE=$(shell date '+%Y-%m-%d-%H:%M:%S')
IMAGE_NAME := "troven/cargo"

default: test

help:
	@echo 'Management commands for cargo:'
	@echo
	@echo 'Usage:'
	@echo '    make build           Compile the project.'
	@echo '    make install         Compile the project and install cargo to $$GOPATH/bin.'
	@echo '    make get-deps        runs dep ensure, mostly used for CI.'
	@echo '    make build-docker    Compile optimized for Docker linux (scratch / alpine).'
	@echo '    make image           Build final docker image with just the go binary inside'
	@echo '    make tag             Tag image created by package with latest, git commit and version'
	@echo '    make test            Run tests on the source code of the project.'
	@echo '    make test-run        Run tests using cargo executable (must be available in $$PATH).'
	@echo '    make test-write      Run tests using cargo executable, writes files.'
	@echo '    make push            Push tagged images to registry'
	@echo '    make clean           Clean the directory tree.'
	@echo

build:
	@echo "building ${BIN_NAME} ${VERSION}$(GIT_DIRTY)"
	@go build -ldflags "-X github.com/troven/cargo/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY} -X github.com/troven/cargo/version.BuildDate=${BUILD_DATE}" \
		-o bin/${BIN_NAME} github.com/troven/cargo/cmd/cargo

install: build
	@echo "installing ${BIN_NAME} ${VERSION}$(GIT_DIRTY)"
	@cp bin/${BIN_NAME} ${GOPATH}/bin/${BIN_NAME}

get-deps:
	dep ensure

build-docker:
	@echo "building ${BIN_NAME} ${VERSION}$(GIT_DIRTY) for Docker"
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
		-ldflags '-X github.com/troven/cargo/version.GitCommit=${GIT_COMMIT}${GIT_DIRTY} -X github.com/troven/cargo/version.BuildDate=${BUILD_DATE}' \
		-o bin/${BIN_NAME}_docker github.com/troven/cargo/cmd/cargo

image: build-docker
	@echo "building image ${BIN_NAME} ${VERSION} $(GIT_COMMIT)"
	docker build -t $(IMAGE_NAME):local .

tag: 
	@echo "Tagging: latest ${VERSION} $(GIT_COMMIT)"
	docker tag $(IMAGE_NAME):local $(IMAGE_NAME):$(GIT_COMMIT)
	docker tag $(IMAGE_NAME):local $(IMAGE_NAME):${VERSION}
	docker tag $(IMAGE_NAME):local $(IMAGE_NAME):latest

push: tag
	@echo "Pushing docker image to registry: latest ${VERSION} $(GIT_COMMIT)"
	docker push $(IMAGE_NAME):$(GIT_COMMIT)
	docker push $(IMAGE_NAME):${VERSION}
	docker push $(IMAGE_NAME):latest

clean:
	@rm -f bin/${BIN_NAME}
	@rm -f bin/${BIN_NAME}_docker
	@rm -rf test/build/

test:
	go test ./...

test-run: export FOO_BAR=kek
test-run:
	cargo run --dry-run \
		--context test/cargo.yaml \
		--context App=test/app.json \
		--context Friends=test/friends.yaml \
		test/cargo test/build

test-write: export FOO_BAR=kek
test-write:
	cargo run \
		--context test/cargo.yaml \
		--context App=test/app.json \
		--context Friends=test/friends.yaml \
		test/cargo test/build

	diff -r test/build test/build_expected


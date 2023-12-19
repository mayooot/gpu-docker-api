BINARY = gpu-docker-api
GOARCH = amd64

BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
VERSION?=?
COMMIT=$(shell git rev-parse HEAD)
GO_VERSION=$(shell go env GOVERSION)
BUILD_TIME=$(shell date +%Y-%m-%d-%H-%M-%S)

GITHUB_USER = mayooot
CURRENT_DIR =$(shell pwd)
BUILD_DIR=${CURRENT_DIR}/cmd/${BINARY}
BIN_DIR=${CURRENT_DIR}/bin

LDFLAGS = -ldflags "-X main.BRANCH=${BRANCH} -X main.VERSION=${VERSION} -X main.COMMIT=${COMMIT} -X main.GO_VERSION=${GO_VERSION} -X main.BUILD_TIME=${BUILD_TIME}"

linux:
	cd ${BUILD_DIR}; \
	GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BIN_DIR}/${BINARY}-linux-${GOARCH} . ; \
	cd - >/dev/null

darwin:
	cd ${BUILD_DIR}; \
	GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BIN_DIR}/${BINARY}-darwin-${GOARCH} . ; \
	cd - >/dev/null

windows:
	cd ${BUILD_DIR}; \
	GOOS=windows GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BIN_DIR}/${BINARY}-windows-${GOARCH}.exe . ; \
	cd - >/dev/null

clean:
	- rm -f ${BIN_DIR}/*

fmt:
	gofmt -l -w .

imports:
	goimports-reviser --rm-unused -local github.com/${GITHUB_USER}/${BINARY} -format ./...

.PHONY: linux darwin windows clean fmt imports
BINARY = gpu-docker-api
GOARCH = amd64

GITHUB_USER = mayooot

CURRENT_DIR =$(shell pwd)
BUILD_DIR=${CURRENT_DIR}/cmd/${BINARY}
BIN_DIR=${CURRENT_DIR}/bin

LDFLAGS = -ldflags "-w -s"

build:
	@cd ${BUILD_DIR}; \
	GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BIN_DIR}/${BINARY}-linux-${GOARCH} . ; \
	cd - >/dev/null

fmt:
	gofmt -l -w .

imports:
	goimports -l -w -local github.com/${GITHUB_USER}/${BINARY} .

.PHONY: linux fmt imports
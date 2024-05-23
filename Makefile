BUILD_OPTION=-ldflags="-s -w"
BUILD_NAME=solizard
SOURCE=./cmd/main.go
INSTALL_PATH=/data
BUILD_PATH=./bin

.PHONY: build clean build-darwin build-linux
default: build
all: clean build build-darwin build-linux
build:
	go build ${BUILD_OPTION} -o ${BUILD_PATH}/${BUILD_NAME} ${SOURCE}
build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ${BUILD_OPTION} -o ${BUILD_PATH}/${BUILD_NAME} ${SOURCE}
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${BUILD_OPTION} -o ${BUILD_PATH}/${BUILD_NAME} ${SOURCE}
clean:
	rm -rf ${BUILD_PATH}
	go clean
#git_hash := $(shell git rev-parse --short HEAD || echo 'development')
#version = ${git_hash}
version = 0.1.0

# Get current date time in UTC
current_time = $(shell date -u +"%Y-%m-%dT%H:%M:%S%Z")

# Add linker flags
linker_flags = '-s -w -X main.buildTime=${current_time} -X main.version=${version}'

# Build binaries for current OS and Linux
build:
	@echo "Building binaries..."
	GOOS=darwin GOARCH=arm64 go build -ldflags=${linker_flags} -o=./build/facedetect.darwin.arm64 .
	GOOS=linux GOARCH=amd64 go build -ldflags=${linker_flags} -o=./build/facedetect.linux.amd64 .

clean:
	rm -rf build/*

package: build
	tar --directory build -czvf ./build/facedetect-${version}.darwin.arm64.tar.gz facedetect.darwin.arm64
	tar --directory build -czvf ./build/facedetect-${version}.linux.amd64.tar.gz facedetect.linux.amd64

test:
	go test -v

.PHONY: build clean package test

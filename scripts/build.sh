#!/bin/sh

mkdir -p bin
BUILD_FLAGS="${BUILD_FLAGS} -X github.com/yusing/go-proxy/pkg.version=${VERSION}"
echo building GoDoxy version "${VERSION}", build flags \""${BUILD_FLAGS}"\"
go build -ldflags "${BUILD_FLAGS}" -pgo=auto -o bin/godoxy ./cmd

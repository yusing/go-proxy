#!/bin/sh

mkdir -p bin
echo building GoDoxy version "${VERSION}", build flags \""${BUILD_FLAGS}"\"
go build -ldflags "${BUILD_FLAGS}" -pgo=auto -o bin/godoxy ./cmd

#!/bin/sh

mkdir -p bin
echo building go-proxy version ${VERSION}, build flags \"${BUILD_FLAGS}\"
go build -ldflags "${BUILD_FLAGS}" -pgo=auto -o bin/go-proxy ./cmd

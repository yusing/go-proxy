#!/bin/sh
mkdir -p bin
CGO_ENABLED=0 GOOS=linux go build -o bin/go-proxy || exit 1

if [ "$1" = "up" ]; then
    docker compose up -d --build app && \
    docker compose logs -f
fi

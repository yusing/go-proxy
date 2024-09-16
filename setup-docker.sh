#!/bin/sh

set -e
if [ -z "$BRANCH" ]; then
    BRANCH="v0.5"
fi
BASE_URL="https://github.com/yusing/go-proxy/raw/${BRANCH}"
mkdir -p go-proxy
cd go-proxy
mkdir -p config
mkdir -p certs
[ -f compose.yml ] || wget -cO - ${BASE_URL}/compose.example.yml > compose.yml
[ -f config/config.yml ] || wget -cO - ${BASE_URL}/config.example.yml > config/config.yml
[ -f config/providers.yml ] || touch config/providers.yml
VERSION ?= $(shell git describe --tags --abbrev=0)
BUILD_FLAGS ?= -s -w
BUILD_DATE ?= $(shell date -u +'%Y%m%d-%H%M')
export VERSION
export BUILD_FLAGS
export CGO_ENABLED = 0
export GOOS = linux

.PHONY: all setup build test up restart logs get debug run archive repush rapid-crash debug-list-containers

all: debug

build:
	scripts/build.sh

test:
	GODOXY_TEST=1 go test ./internal/...

up:
	docker compose up -d

restart:
	docker compose restart -t 0

logs:
	docker compose logs -f

get:
	go get -u ./cmd && go mod tidy

debug:
	GODOXY_DEBUG=1 BUILD_FLAGS="" make run

debug-trace:
	GODOXY_TRACE=1 make debug

profile:
	GODEBUG=gctrace=1 make debug

run: build
	sudo setcap CAP_NET_BIND_SERVICE=+eip bin/godoxy
	bin/godoxy

mtrace:
	bin/godoxy debug-ls-mtrace > mtrace.json

rapid-crash:
	sudo docker run --restart=always --name test_crash -p 80 debian:bookworm-slim /bin/cat &&\
	sleep 3 &&\
	sudo docker rm -f test_crash

debug-list-containers:
	bash -c 'echo -e "GET /containers/json HTTP/1.0\r\n" | sudo netcat -U /var/run/docker.sock | tail -n +9 | jq'

ci-test:
	mkdir -p /tmp/artifacts
	act -n --artifact-server-path /tmp/artifacts -s GITHUB_TOKEN="$$(gh auth token)"

cloc:
	cloc --not-match-f '_test.go$$' cmd internal pkg

push-docker-io:
	BUILDER=build docker buildx build \
		--platform linux/arm64,linux/amd64 \
		-f Dockerfile.dev \
		-t docker.io/yusing/godoxy-nightly \
		--build-arg VERSION="${VERSION}-nightly-${BUILD_DATE}" \
		--push .

build-docker:
	docker build -t godoxy-nightly \
		--build-arg VERSION="${VERSION}-nightly-${BUILD_DATE}" .
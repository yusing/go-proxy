VERSION ?= $(shell git describe --tags --abbrev=0)
BUILD_FLAGS ?= -s -w -X github.com/yusing/go-proxy/pkg.version=${VERSION}
export VERSION
export BUILD_FLAGS
export CGO_ENABLED = 0
export GOOS = linux

.PHONY: all setup build test up restart logs get debug run archive repush rapid-crash debug-list-containers

all: debug

build:
	scripts/build.sh

test:
	GOPROXY_TEST=1 go test ./internal/...

up:
	docker compose up -d

restart:
	docker compose restart -t 0

logs:
	docker compose logs -f

get:
	go get -u ./cmd && go mod tidy

debug:
	make build
	sudo GOPROXY_DEBUG=1 bin/go-proxy

debug-trace:
	make build
	sudo GOPROXY_DEBUG=1 GOPROXY_TRACE=1 bin/go-proxy

profile:
	GODEBUG=gctrace=1 make build
	sudo GOPROXY_DEBUG=1 bin/go-proxy

mtrace:
	bin/go-proxy debug-ls-mtrace > mtrace.json

run:
	make build && sudo bin/go-proxy

archive:
	git archive HEAD -o ../go-proxy-$$(date +"%Y%m%d%H%M").zip

repush:
	git reset --soft HEAD^
	git add -A
	git commit -m "repush"
	git push gitlab dev --force

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
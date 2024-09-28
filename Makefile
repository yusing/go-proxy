BUILD_FLAG ?= -s -w

.PHONY: all setup build test up restart logs get debug run archive repush rapid-crash debug-list-containers

all: debug

setup:
	mkdir -p config certs
	[ -f config/config.yml ] || cp config.example.yml config/config.yml
	[ -f config/providers.yml ] || touch config/providers.yml

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux \
		go build -ldflags '${BUILD_FLAG}' -pgo=auto -o bin/go-proxy ./cmd

test:
	go test ./internal/...

up:
	docker compose up -d

restart:
	docker compose restart -t 0

logs:
	docker compose logs -f

get:
	cd cmd && go get -u && go mod tidy && cd ..

debug:
	make BUILD_FLAG="" build && sudo GOPROXY_DEBUG=1 bin/go-proxy

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
	sudo docker run --restart=always --name test_crash debian:bookworm-slim /bin/cat &&\
	sleep 3 &&\
	sudo docker rm -f test_crash

debug-list-containers:
	bash -c 'echo -e "GET /containers/json HTTP/1.0\r\n" | sudo netcat -U /var/run/docker.sock | tail -n +9 | jq'

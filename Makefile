.PHONY: all build up quick-restart restart logs get udp-server

all: build quick-restart logs

init-config:
	mkdir -p config certs
	[ -f config/config.yml ] || cp config.example.yml config/config.yml
	[ -f config/providers.yml ] || touch config/providers.yml

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -pgo=auto -o bin/go-proxy src/go-proxy/*.go

up:
	docker compose up -d --build go-proxy

restart:
	docker kill go-proxy
	docker compose up -d go-proxy

logs:
	tail -f log/go-proxy.log

get:
	go get -d -u ./src/go-proxy

udp-server:
	docker run -it --rm \
		-p 9999:9999/udp \
		--label proxy.test-udp.scheme=udp \
		--label proxy.test-udp.port=20003:9999 \
		--network data_default \
		--name test-udp \
		$$(docker build -q -f udp-test-server.Dockerfile .)

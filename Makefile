.PHONY: build up restart logs get test-udp-container

all: build up logs

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -o bin/go-proxy src/go-proxy/*.go

up:
	docker compose up -d --build go-proxy

restart:
	docker compose down -t 0
	docker compose up -d

logs:
	docker compose logs -f

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
.PHONY: all build up quick-restart restart logs get udp-server

all: build quick-restart logs

setup:
	mkdir -p config certs
	[ -f config/config.yml ] || cp config.example.yml config/config.yml
	[ -f config/providers.yml ] || touch config/providers.yml

setup-codemirror:
	wget https://codemirror.net/5/codemirror.zip
	unzip codemirror.zip
	rm codemirror.zip
	mv codemirror-* templates/codemirror

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -pgo=auto -o bin/go-proxy src/go-proxy/*.go

up:
	docker compose up -d --build app

restart:
	docker kill go-proxy
	docker compose up -d app

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

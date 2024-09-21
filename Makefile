.PHONY: all build up quick-restart restart logs get udp-server

all: build quick-restart logs

setup:
	mkdir -p config certs
	[ -f config/config.yml ] || cp config.example.yml config/config.yml
	[ -f config/providers.yml ] || touch config/providers.yml

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -pgo=auto -o bin/go-proxy github.com/yusing/go-proxy

test:
	cd src && go test ./... && cd ..

up:
	docker compose up -d

restart:
	docker compose restart -t 0

logs:
	docker compose logs -f

get:
	cd src && go get -u && go mod tidy && cd ..

debug:
	make build && sudo GOPROXY_DEBUG=1 bin/go-proxy

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
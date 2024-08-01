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
	mkdir -p templates
	mv codemirror-* templates/codemirror

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -pgo=auto -o bin/go-proxy github.com/yusing/go-proxy

test:
	cd src && go test && cd ..

up:
	docker compose up -d

restart:
	docker compose restart -t 0

logs:
	docker compose logs -f

get:
	cd src && go get -u && go mod tidy && cd ..

debug:
	make build && GOPROXY_DEBUG=1 bin/go-proxy

archive:
	git archive HEAD -o ../go-proxy-$$(date +"%Y%m%d%H%M").zip

repush:
	git reset --soft HEAD^
	git add -A
	git commit -m "repush"
	git push gitlab dev --force
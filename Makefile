export VERSION ?= $(shell git describe --tags --abbrev=0)
export BUILD_DATE ?= $(shell date -u +'%Y%m%d-%H%M')
export GOOS = linux

LDFLAGS = -X github.com/yusing/go-proxy/pkg.version=${VERSION}

ifeq ($(trace), 1)
	GODOXY_TRACE ?= 1
endif

ifeq ($(debug), 1)
	CGO_ENABLED = 0
	GODOXY_DEBUG = 1
  BUILD_FLAGS = ''
else ifeq ($(pprof), 1)
	CGO_ENABLED = 1
	GODEBUG = gctrace=1 inittrace=1 schedtrace=3000
	GORACE = log_path=logs/pprof strip_path_prefix=$(shell pwd)/
  BUILD_FLAGS = -race -gcflags=all='-N -l' -tags pprof
	DOCKER_TAG = pprof
	VERSION += -pprof
else
	CGO_ENABLED = 0
	LDFLAGS += -s -w
  BUILD_FLAGS = -pgo=auto -tags production
	DOCKER_TAG = latest
endif

BUILD_FLAGS += -ldflags='$(LDFLAGS)'

export CGO_ENABLED
export GODOXY_DEBUG
export GODOXY_TRACE
export GODEBUG
export GORACE
export BUILD_FLAGS
export DOCKER_TAG

test:
	GODOXY_TEST=1 go test ./internal/...

get:
	go get -u ./cmd && go mod tidy

build:
	mkdir -p bin
	go build ${BUILD_FLAGS} -o bin/godoxy ./cmd
	if [ $(shell id -u) -eq 0 ]; \
		then setcap CAP_NET_BIND_SERVICE=+eip bin/godoxy; \
		else sudo setcap CAP_NET_BIND_SERVICE=+eip bin/godoxy; \
	fi

run:
	[ -f .env ] && godotenv -f .env go run ${BUILD_FLAGS} ./cmd

mtrace:
	bin/godoxy debug-ls-mtrace > mtrace.json

rapid-crash:
	docker run --restart=always --name test_crash -p 80 debian:bookworm-slim /bin/cat &&\
	sleep 3 &&\
	docker rm -f test_crash

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
		-f Dockerfile \
		-t docker.io/yusing/godoxy-nightly:${DOCKER_TAG} \
		-t docker.io/yusing/godoxy-nightly:${VERSION}-${BUILD_DATE} \
		--build-arg VERSION="${VERSION}-nightly-${BUILD_DATE}" \
		--build-arg BUILD_FLAGS="${BUILD_FLAGS}" \
		--push .

build-docker:
	docker build -t godoxy-nightly \
		--build-arg VERSION="${VERSION}-nightly-${BUILD_DATE}" \
		--build-arg BUILD_FLAGS="${BUILD_FLAGS}" \
		.

gen-schema-single:
	typescript-json-schema --noExtraProps --required --skipLibCheck --tsNodeRegister=true -o schemas/${OUT} schemas/${IN} ${CLASS}

gen-schema:
	make IN=config/config.ts \
			CLASS=Config \
			OUT=config.schema.json \
			gen-schema-single
	make IN=providers/routes.ts \
			CLASS=Routes \
			OUT=routes.schema.json \
			gen-schema-single
	make IN=middlewares/middleware_compose.ts \
			CLASS=MiddlewareCompose \
			OUT=middleware_compose.schema.json \
			gen-schema-single
	make IN=docker.ts \
			CLASS=DockerRoutes \
			OUT=docker_routes.schema.json \
			gen-schema-single

push-github:
	git push origin $(shell git rev-parse --abbrev-ref HEAD)
# Stage 1: Builder
FROM golang:1.23.1-alpine AS builder
RUN apk add --no-cache tzdata make

WORKDIR /src

# Only copy go.mod and go.sum initially for better caching
COPY go.mod go.sum /src/

# Utilize build cache
RUN --mount=type=cache,target="/go/pkg/mod" \
    go mod download -x

ENV GOCACHE=/root/.cache/go-build

ARG VERSION
ENV VERSION=${VERSION}

COPY scripts /src/scripts
COPY Makefile /src/

RUN --mount=type=cache,target="/go/pkg/mod" \
    --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=bind,src=cmd,dst=/src/cmd \
    --mount=type=bind,src=internal,dst=/src/internal \
    --mount=type=bind,src=pkg,dst=/src/pkg \
    make build && \
    mkdir -p /app/error_pages /app/certs && \
    mv bin/go-proxy /app/go-proxy

# Stage 2: Final image
FROM scratch

LABEL maintainer="yusing@6uo.me"
LABEL proxy.exclude=1

# copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# copy binary
COPY --from=builder /app /app

# copy schema directory
COPY schema/ /app/schema/

# copy certs
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

ENV DOCKER_HOST=unix:///var/run/docker.sock
ENV GOPROXY_DEBUG=0

EXPOSE 80
EXPOSE 8888
EXPOSE 443

WORKDIR /app

CMD ["/app/go-proxy"]

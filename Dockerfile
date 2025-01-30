# trunk-ignore-all(checkov/CKV_DOCKER_3)
# Stage 1: Builder
FROM golang:1.23.5-alpine AS builder
HEALTHCHECK NONE

# package version does not matter
# trunk-ignore(hadolint/DL3018)
RUN apk add --no-cache make

WORKDIR /src

# Only copy go.mod and go.sum initially for better caching
COPY go.mod go.sum /src/

# Utilize build cache
RUN --mount=type=cache,target="/go/pkg/mod" \
    go mod download -x

ENV GOCACHE=/root/.cache/go-build

COPY Makefile /src/
COPY cmd /src/cmd
COPY internal /src/internal
COPY pkg /src/pkg

ARG VERSION
ENV VERSION=${VERSION}

ARG BUILD_FLAGS
ENV BUILD_FLAGS=${BUILD_FLAGS}

RUN --mount=type=cache,target="/go/pkg/mod" \
    --mount=type=cache,target="/root/.cache/go-build" \
    make build && \
    mkdir -p /app/error_pages /app/certs && \
    mv bin/godoxy /app/godoxy

# Stage 2: Final image
FROM alpine:3

LABEL maintainer="yusing@6uo.me"
LABEL proxy.exclude=1

# trunk-ignore(hadolint/DL3018)
RUN apk add --no-cache tzdata ca-certificates runuser libcap-setcap socat

# copy binary
COPY --from=builder /app /app

# copy startup script
COPY scripts/docker-start.sh /app/docker-start.sh

RUN chmod +x /app/docker-start.sh

# copy example config
COPY config.example.yml /app/config/config.yml

ENV SOCKET_FORK=/app/forked.sock
ENV DOCKER_HOST=unix://${SOCKET_FORK}
ENV GODOXY_DEBUG=0

ENV PUID=1002
ENV PGID=1002

EXPOSE 80
EXPOSE 8888
EXPOSE 443

WORKDIR /app

ENTRYPOINT [ "/app/docker-start.sh" ]
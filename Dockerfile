# Stage 1: Builder
FROM golang:1.23.4-alpine AS builder
HEALTHCHECK NONE

# package version does not matter
# trunk-ignore(hadolint/DL3018)
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
COPY cmd /src/cmd
COPY internal /src/internal
COPY pkg /src/pkg

RUN --mount=type=cache,target="/go/pkg/mod" \
    --mount=type=cache,target="/root/.cache/go-build" \
    make build && \
    mkdir -p /app/error_pages /app/certs && \
    mv bin/godoxy /app/godoxy

# Stage 2: Final image
FROM scratch

LABEL maintainer="yusing@6uo.me"
LABEL proxy.exclude=1

# copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# copy binary
COPY --from=builder /app /app

# copy example config
COPY config.example.yml /app/config/config.yml

# copy certs
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

# copy schema
COPY schema /app/schema

ENV DOCKER_HOST=unix:///var/run/docker.sock
ENV GODOXY_DEBUG=0

EXPOSE 80
EXPOSE 8888
EXPOSE 443

WORKDIR /app

CMD ["/app/godoxy"]
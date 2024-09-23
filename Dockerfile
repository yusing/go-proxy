FROM golang:1.23.1-alpine AS builder
RUN apk add --no-cache tzdata
COPY src /src
ENV GOCACHE=/root/.cache/go-build
WORKDIR /src
RUN --mount=type=cache,target="/go/pkg/mod" \
    --mount=type=cache,target="/root/.cache/go-build" \
    go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -pgo=auto -o go-proxy github.com/yusing/go-proxy

FROM scratch

LABEL maintainer="yusing@6uo.me"

# copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# copy binary
COPY --from=builder /src/go-proxy /app/
COPY schema/ /app/schema

# copy cert required for setup
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

ENV DOCKER_HOST=unix:///var/run/docker.sock
ENV GOPROXY_DEBUG=0

EXPOSE 80
EXPOSE 8888
EXPOSE 443

WORKDIR /app
CMD ["/app/go-proxy"]
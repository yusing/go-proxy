FROM golang:1.23.1-alpine AS builder
COPY src /src
ENV GOCACHE=/root/.cache/go-build
WORKDIR /src
RUN --mount=type=cache,target="/go/pkg/mod" \
    --mount=type=cache,target="/root/.cache/go-build" \
    go mod download
    CGO_ENABLED=0 GOOS=linux go build -pgo=auto -o go-proxy github.com/yusing/go-proxy

FROM alpine:3.20

LABEL maintainer="yusing@6uo.me"

RUN apk add --no-cache tzdata
# copy binary
COPY --from=builder /src/go-proxy /app/
COPY schema/ /app/schema

RUN chmod +x /app/go-proxy
ENV DOCKER_HOST=unix:///var/run/docker.sock
ENV GOPROXY_DEBUG=0

EXPOSE 80
EXPOSE 8888
EXPOSE 443

WORKDIR /app
CMD ["/app/go-proxy"]
FROM alpine:latest AS codemirror
RUN apk add --no-cache unzip wget make
COPY Makefile .
RUN make setup-codemirror

FROM golang:1.22.2-alpine as builder
COPY src/ /src
COPY go.mod go.sum /src/go-proxy
WORKDIR /src/go-proxy
RUN --mount=type=cache,target="/go/pkg/mod" \
    go mod download

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target="/go/pkg/mod" \
    --mount=type=cache,target="/root/.cache/go-build" \
    CGO_ENABLED=0 GOOS=linux go build -pgo=auto -o go-proxy

FROM alpine:latest

LABEL maintainer="yusing@6uo.me"

RUN apk add --no-cache tzdata
RUN mkdir -p /app/templates
COPY --from=codemirror templates/codemirror/ /app/templates/codemirror
COPY templates/ /app/templates
COPY schema/ /app/schema
COPY --from=builder /src/go-proxy /app/

RUN chmod +x /app/go-proxy
ENV DOCKER_HOST unix:///var/run/docker.sock
ENV GOPROXY_DEBUG 0

EXPOSE 80
EXPOSE 8080
EXPOSE 443
EXPOSE 8443

WORKDIR /app
CMD ["/app/go-proxy"]
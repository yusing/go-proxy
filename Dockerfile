FROM golang:1.22.1 as builder

COPY go.mod /app/go.mod
COPY src/ /app/src
COPY Makefile /app
WORKDIR /app
RUN make get
RUN make build

FROM alpine:latest

LABEL maintainer="yusing@6uo.me"

RUN apk add --no-cache tzdata
COPY --from=builder /app/bin/go-proxy /app/
COPY templates/ /app/templates
COPY schema/ /app/schema

RUN chmod +x /app/go-proxy
ENV DOCKER_HOST unix:///var/run/docker.sock
ENV GOPROXY_DEBUG 0

EXPOSE 80
EXPOSE 8080
EXPOSE 443
EXPOSE 8443

WORKDIR /app
CMD ["/app/go-proxy"]
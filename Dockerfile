# Stage 1: Builder
FROM golang:1.23.1-alpine AS builder
RUN apk add --no-cache tzdata

WORKDIR /src

# Only copy go.mod and go.sum initially for better caching
COPY src/go.mod src/go.sum ./

# Utilize build cache
RUN --mount=type=cache,target="/go/pkg/mod" \
    go mod download

# Now copy the remaining files
COPY src/ ./

# Build the application with better caching
RUN --mount=type=cache,target="/go/pkg/mod" \
    --mount=type=cache,target="/root/.cache/go-build" \
    CGO_ENABLED=0 GOOS=linux go build -pgo=auto -o go-proxy ./

# Stage 2: Final image
FROM scratch

LABEL maintainer="yusing@6uo.me"

# copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# copy binary
COPY --from=builder /src/go-proxy /app/

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
